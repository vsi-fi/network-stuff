/* Subscribe receive link flaps as they happen 
<vsimola@hc.nrec> */
package main

import (
	"context"
	"io"
	"sync"
	"log"
	"fmt"
	"strings"
	"encoding/json"
	"github.com/openconfig/gnmi/proto/gnmi"
	//"github.com/davecgh/go-spew/spew"
	"encoding/base64"
	//"os"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	//"github.com/openconfig/ygot/protomap"
	//gpb "github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"flag"
	"strconv"
	//"reflect"
	"google.golang.org/protobuf/encoding/protojson"
)

type TelemetryEvent struct {
    Device     string
    Interface  string
    Timestamp  string
    OperStatus string
}

func ExtractInterfaceName(data map[string]interface{}) (string, bool) {
	update, ok := data["update"].(map[string]interface{})
	if !ok {
		return "", false
	}

	prefix, ok := update["prefix"].(map[string]interface{})
	if !ok {
		return "", false
	}
	elemList, ok := prefix["elem"].([]interface{})
	if !ok {
		return "", false
	}
	for _, elemItem := range elemList {
		elemMap, ok := elemItem.(map[string]interface{})
		if !ok {
			continue
		}
		if keyMap, ok := elemMap["key"].(map[string]interface{}); ok {
			if name, ok := keyMap["name"].(string); ok {
				return name, true
			}
		}
	}
	return "", false
}
func ExtractEventInfo(data map[string]interface{}) (name string, timestamp string, operstatus string, found bool) {
	updateMap, ok := data["update"].(map[string]interface{})
	if !ok {
		return "", "", "", false
	}

	timestamp, _ = updateMap["timestamp"].(string)

	// Extract name from prefix
	if prefix, ok := updateMap["prefix"].(map[string]interface{}); ok {
		if elemList, ok := prefix["elem"].([]interface{}); ok {
			for _, elemItem := range elemList {
				elemMap, ok := elemItem.(map[string]interface{})
				if !ok {
					continue
				}
				if keyMap, ok := elemMap["key"].(map[string]interface{}); ok {
					if n, ok := keyMap["name"].(string); ok {
						name = n
					}
				}
			}
		}
	}

	// Iterate over the inner update list
	updateList, ok := updateMap["update"].([]interface{})
	if ok {
		for _, updateItem := range updateList {
			updateEntry, ok := updateItem.(map[string]interface{})
			if !ok {
				continue
			}

			// Check if path contains "oper-status"
			path, ok := updateEntry["path"].(map[string]interface{})
			if !ok {
				continue
			}

			elemList, ok := path["elem"].([]interface{})
			if !ok {
				continue
			}

			for _, elem := range elemList {
				elemMap, ok := elem.(map[string]interface{})
				if !ok {
					continue
				}
				if elemMap["name"] == "oper-status" {
					if valMap, ok := updateEntry["val"].(map[string]interface{}); ok {
						if jsonValBase64, ok := valMap["json_val"].(string); ok {
							if decoded, err := base64.StdEncoding.DecodeString(jsonValBase64); err == nil {
								var status string
								if err := json.Unmarshal(decoded, &status); err == nil {
									operstatus = status
								}
							}
						}
					}
				}
			}
		}
	}

	found = name != "" || operstatus != ""
	return name, timestamp, operstatus, found
}


func subscribeResponseToMap(resp *gnmi.SubscribeResponse) (map[string]interface{}, error) {
	// Marshal SubscribeResponse to JSON
	marshaler := protojson.MarshalOptions{
		EmitUnpopulated: true,  // include zero values
		UseProtoNames:   true,  // use proto field names (not camelCase)
	}
	jsonBytes, err := marshaler.Marshal(resp)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON into map
	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func SubscribeToDevice(device string, port int, path string, ch chan <- TelemetryEvent, wg *sync.WaitGroup) error {
	defer wg.Done()
	// Define the Juniper device's address and port
        serverAddr := device + ":" + strconv.Itoa(port) // Replace with your device's IP and gRPC port

        // Create a gRPC connection
        conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
        if err != nil {
            log.Fatalf("Failed to connect to server: %v", err)
        }
        defer conn.Close()

        client := pb.NewGNMIClient(conn)
	// Define a subscription request
	subscribeRequest := &pb.SubscribeRequest{
	    Request: &pb.SubscribeRequest_Subscribe{
	        Subscribe: &pb.SubscriptionList{
	            Prefix: &pb.Path{
	                Elem: []*pb.PathElem{
	                    {Name: path},
	                },
	            },
	            Subscription: []*pb.Subscription{
	                {
	                    Path: &pb.Path{
	                        Elem: []*pb.PathElem{
					//{Name: "interface", Key: map[string]string{"name": "et-0/0/12:2"}},
					{Name: "interface"},
	                            {Name: "state"},
	                            {Name: "oper-status"},
				},
	                    },
	                    //Mode:           pb.SubscriptionMode_SAMPLE,
	                    Mode:           pb.SubscriptionMode_ON_CHANGE,
	                    //SampleInterval: 30 * 1e9, // 10 seconds in nanoseconds
	                },
	            },
	            Mode: pb.SubscriptionList_STREAM,
	        },
	    },
	}

	// Create a context
	ctx := context.Background()

	// Get the subscription stream
	stream, err := client.Subscribe(ctx)
	if err != nil {
	    log.Fatalf("Error creating subscription stream: %v", err)
	}

	// Send the SubscribeRequest through the stream
	err = stream.Send(subscribeRequest)
	if err != nil {
	    log.Fatalf("Error sending subscription request: %v", err)
	}

	log.Println("Subscribed successfully. Waiting for telemetry data...")

	// Receive and process responses
	for {
	    response, err := stream.Recv()
	    if err == io.EOF {
	        break
	    }
	    if err != nil {
	        log.Fatalf("Error receiving response: %v", err)
	    }
	    //log.Printf("Received telemetry data: %+v\n", response)
	    //spew.Dump(subscribeResponseToMap(response))
	    data,_ := subscribeResponseToMap(response)
	    //ifacename, found := ExtractInterfaceName(data)
	    ifacename, timestamp, operstatus, found := ExtractEventInfo(data)
	    if  found  {
		//spew.Dump(subscribeResponseToMap(response))
		ch  <- TelemetryEvent{
			Device: device,
			Interface: ifacename,
			Timestamp: timestamp,
			OperStatus: operstatus,
		}
		//fmt.Println("Device", device, "Interface: ", ifacename, "Operstatus:", operstatus, "Timestamp:", timestamp)
		}
	    //fmt.Println(reflect.TypeOf(response))
	//os.Exit(0)
	}


	return nil
}

func main() {
	devices_ptr := flag.String("devices", "", "A list,of,devices for which we want to create a subscription for.")
	port_ptr := flag.Int("port", 8011, "Port on which the telemetry thingy is listening on the device.")
	path_ptr := flag.String("path", "interfaces","Path from which we want to subscribe for data.")
	flag.Parse()

	var wg sync.WaitGroup
	devices := strings.Split(*devices_ptr, ",")
	resultsCh := make(chan TelemetryEvent)
	for _,device := range devices {
		device := device  // Shadowing to avoid closure capture issue
		wg.Add(1)
		go func(dev string) {
			defer wg.Done()
			err := SubscribeToDevice(device, *port_ptr, *path_ptr, resultsCh, &wg)
			if err  != nil {
				fmt.Println("Failed while trying to subsribe to ", dev, err)
			}
		}(device)
	}
	for event := range resultsCh {
		fmt.Println("MAIN Device", event.Device, "Interface: ", event.Interface, "Operstatus:", event.OperStatus, "Timestamp:", event.Timestamp)
	}
	wg.Wait()
}
