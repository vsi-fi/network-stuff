/* testing juniper rest API for collecting data
<vsimola@hc.nrec>*/
package main

import (
        "fmt"
        "bytes"
        "io/ioutil"
        "net/http"
        //"strings"
        //"log"
        "encoding/json"
        //"reflect"
        "crypto/tls"
        "regexp"
        "time"
        "encoding/base64"
)

const USER = "restro"
const PASS = "salaisuusSecret"

const URL = "https://device.name:3443/rpc"

func SkipVerification() (*tls.Config, error) {
                return &tls.Config{InsecureSkipVerify: true}, nil
        }

//Remove beginning/end separators and headers
func Cleanup(jason string) string {
        var re = regexp.MustCompile(".*utf-8|--harqgehabymwiax|--harqgehabymwiax--|--")
        j := re.ReplaceAllString(jason, "")
        return(j)
}

func ParseJason(jason string) map[string]any {
        var data map[string]any
        err := json.Unmarshal([]byte(jason), &data)
        if err != nil {
                fmt.Println("Failed to parse json", err)
        }
        return(data)
}

func GenerateResponse(data interface{}) {
        switch v := data.(type) {
        case map[string]interface{}:
                for key,value := range v {
                        if key != "data" {
                                fmt.Println("1 Key %s", key)
                        }
                        GenerateResponse(value)
                }
        case []interface{}:
                for i, value := range v {
                        if i != 0 {
                                fmt.Println("2 Key %s",i)
                        }
                        GenerateResponse(value)
                }
        default:
                fmt.Println("VALUE %v", v)
        }

}func MakeRequest(api string) string {
        fmt.Println("Trying to make a http request to " + URL)
        tlsConf, err := SkipVerification()

        query := "<get-interface-information><interface-name>xe-0/0/10</interface-name></get-interface-information>"
        //query := "<get-system-core-dumps></get-system-core-dumps>" //this does not generate proper json on some platforms
        //query := "<get-environment-fpc-information></get-environment-fpc-information>"
        req, err  := http.NewRequest("POST", api, bytes.NewBuffer([]byte(query)))
        auth := USER + ":" + PASS
        encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))

        req.Header.Set("Content-Type", "application/xml")
        req.Header.Set("Accept", "application/json")
        req.Header.Set("Authorization", "Basic "+encodedAuth)

        client := &http.Client{
                Timeout: time.Second * 10,
                Transport: &http.Transport{TLSClientConfig: tlsConf},
        }
        resp, err := client.Do(req)
        if err != nil {
                fmt.Println("Error sending request:", err)
        }
        defer resp.Body.Close()


        if resp.StatusCode != http.StatusOK {
                fmt.Printf("Error: received status code %d\n", resp.StatusCode)
        }

        responseBody, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                fmt.Println("Error reading response:", err)
        }

        return (Cleanup(string(responseBody)))
}

func main() {
        jason := MakeRequest(URL)
        data := ParseJason(jason)
        GenerateResponse(data)
}

