# junosmator

Run commands on multiple junos boxes at the same time.

## Description

junosmator.pl can be used to:

- Run a set of set-commands on a set of juniper devices
- Run a set of set commands on a single juniper device
- Run both of the above in a dry run (show | compare + commit check + rollback) or real deal (commit confirmed) -style
- Run some random junos commands on single or set of juniper devices 

## Usage
### Access to devices
By default the script tries to ssh into the devices.

This is obviously cumbersome without ssh-keys and ssh-agent. This also means that your keys need to be installed on the devices you plan on managing. 

However, you can get ssh-agent working like so:
Run ssh-agent and copy-paste the output to your shell
```
ssh-agent

```
Then, add your keys and authenticate
```
ssh-add
```
At this point you should have access, assuming that your keys are distributed.
### Inventory file
junosmator.pl needs an inventory file containing "categories" of devices that contain the individual devices.
Inventory file syntax is as follows:
```
[CATEGORY-NAME]
sw-member-switch-1
sw-member-switch-22
[CATEGORY2-NAME]
sw-member-switch-11
sw-member-switch-122
```
### Running commands

To run a random command on a set of devices: 
```
junosmator.pl -i|--inventory $file_containing_inventory_of_devices -f|--filter $string_to_filter_target_groups_for -c|--cmd "$junos_command; $followed_by_maybe_another_one"
```
To get a list of target devices and bail out: 
```
junosmator.pl -i|--inventory $file_containing_inventory_of_devices -h|--hosts
```
To run a set of set-cmds in dry run mode: 
```
junosmator.pl -i|--inventory $file_containing_inventory_of_devices -m|--mode dry_run -t|--template $file_containing_the_commands
```
To run a set of set-cmds and actually try to commit (confirmed): 
```
junosmator.pl -i|--inventory $file_containing_inventory_of_devices -t|--type real_deal -t|--template $file_containing_the_commands
```
**Example**: To execute commands in file "lots_of_cmds", display changes and rollback: 
```
junosmator.pl -i inventory -f ctrl -m dry_run -t lots_of_cmds 
```
**Example**: To check version of category switches: 
```
junosmator.pl -i inventory -c 'show version | match \"EX  Software Suite\"' -f $some_string_that_matches_a_host_category_in_inventory 
```
**Example**: To check version of a single switch: 
```
junosmator.pl -i inventory -sh $host_name -c 'show version'
```
**Example**: To commit a set of of commands to a single switch: 
```
junosmator.pl -i inventory -sh $host_name -t $file_with_commands_in_it -m real_deal
```
**Example**: To review a change on a single switch:
```
junosmator.pl -i inventory -sh $host_name -m dry_run -c "configure; set system domain-name someotherdomain.org; show | compare; rollback;"
```
