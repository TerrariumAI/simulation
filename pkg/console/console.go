package console

import (
	"bufio"
	"context"
	b64 "encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"

	"google.golang.org/grpc/metadata"

	api "github.com/terrariumai/simulation/pkg/api/environment"
)

type command struct {
	ID   string
	Desc string
	Args []string
}

var (
	commands = []command{
		{
			"createAgent",
			"Creates a new agent, owned by your model, at an x,y position.",
			[]string{
				"x:uint",
				"y:uint",
			},
		},
		{
			"getEntity",
			"Gets info for an entity at a given position",
			[]string{
				"id:string",
			},
		},
	}
)

func StartConsole(s api.EnvironmentServer) {
	fmt.Println("This is the environment console. Type 'help' for a list of commands")
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("-> ")
		text, _ := reader.ReadString('\n')
		// convert CRLF to LF
		text = strings.Replace(text, "\n", "", -1)
		words := strings.Fields(text)
		// Make sure something was entered
		if len(words) == 0 {
			continue
		}
		enteredCmd := words[0]

		// Help displays info
		if enteredCmd == "help" {
			for _, otherCommand := range commands {
				fmt.Printf("\t%s %v\n", otherCommand.ID, strings.Join(otherCommand.Args, " "))
				fmt.Printf("\t\t%s\n", otherCommand.Desc)
				fmt.Println("")
			}
		}

		// Parse data
		for _, command := range commands {
			if enteredCmd == command.ID {

				// Make sure all arguments exist
				if len(words)-1 != len(command.Args) {
					fmt.Printf("\tMissing arguments.\n")
					break
				}

				// Create call context
				userinfoJSONString := "{\"id\":\"MOCK-UID\"}"
				userinfoEnc := b64.StdEncoding.EncodeToString([]byte(userinfoJSONString))
				md := metadata.Pairs("x-endpoint-api-userinfo", userinfoEnc)
				ctx := metadata.NewIncomingContext(context.Background(), md)

				// Execute command
				switch enteredCmd {
				case "createAgent":
					x, err := strconv.Atoi(words[1])
					if err != nil || x < 0 {
						fmt.Printf("Error: x must be a positive number")
					}
					y, err := strconv.Atoi(words[2])
					if err != nil || y < 0 {
						fmt.Printf("Error: y must be a positive number")
					}
					resp, err := s.CreateEntity(ctx, &api.CreateEntityRequest{
						Entity: &api.Entity{
							X:       uint32(x),
							Y:       uint32(y),
							ClassID: 1,
							ModelID: "MOCK-MODEL-ID",
						},
					})
					if err != nil {
						fmt.Printf("\t%v\n", err)
						break
					}
					fmt.Printf("\t%v\n", resp)
				case "getEntity":
					id := words[1]
					resp, err := s.GetEntity(ctx, &api.GetEntityRequest{
						Id: id,
					})
					if err != nil {
						fmt.Printf("\t%v\n", err)
					} else {
						fmt.Printf("\t%v\n", resp)
					}
				default:
					fmt.Printf("\tUnrecognized command\n")
				}
			}
		}
	}
}
