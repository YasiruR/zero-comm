package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func Init(name, pubKey, addr string) {
	fmt.Printf("# Agent initialized with following attributes: \n\t- Name: %s\n\t- Endpoint: %s\n\t- Public key: %s\n", name, addr, pubKey)
}

func BasicCommands() {
basicCmds:
	fmt.Printf("# Enter the corresponding number of a command to proceed:\n\t[1] Set recipient\n\t[2] Send a message\nCommand: ")
	reader := bufio.NewReader(os.Stdin)
	cmd, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error: reading command number failed, please try again")
		goto basicCmds
	}

	switch strings.TrimSpace(cmd) {
	case "1":
		fmt.Println(SetRecipient())
	default:
		fmt.Println("Error: invalid command number, please try again")
		goto basicCmds
	}
}

func SetRecipient() (name, endpoint, pubKey string) {
	fmt.Printf("# Enter recipient details:\n")
readName:
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("\tName: ")
	name, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error: reading name failed, please try again")
		goto readName
	}

readEndpoint:
	fmt.Printf("\tEndpoint: ")
	endpoint, err = reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error: reading endpoint failed, please try again")
		goto readEndpoint
	}

readPubKey:
	fmt.Printf("\tPublic key: ")
	pubKey, err = reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error: reading public key failed, please try again")
		goto readPubKey
	}

	return strings.TrimSpace(name), strings.TrimSpace(endpoint), strings.TrimSpace(pubKey)
}
