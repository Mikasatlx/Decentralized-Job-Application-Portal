// Package main implements a simple CLI that can start the http proxy.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	urfave "github.com/urfave/cli/v2"
	"go.dedis.ch/cs438/gui/httpnode"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/peer/impl"
	"go.dedis.ch/cs438/registry/standard"

	"go.dedis.ch/cs438/storage"
	"go.dedis.ch/cs438/storage/file"
	"go.dedis.ch/cs438/storage/inmemory"

	"go.dedis.ch/cs438/transport/udp"
	"golang.org/x/xerrors"
)

const peerAddrMsg = "peer addr: '%s'"

var peerFactory = impl.NewPeer

var (
	// defaultLevel can be changed to set the desired level of the logger
	defaultLevel = zerolog.InfoLevel

	// logout is the logger configuration
	logout = zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}

	log zerolog.Logger
)

func init() {
	if os.Getenv("HTTPLOG") == "warn" {
		defaultLevel = zerolog.WarnLevel
	}

	if os.Getenv("HTTPLOG") == "no" {
		defaultLevel = zerolog.Disabled
	}

	log = zerolog.New(logout).
		Level(defaultLevel).
		With().Timestamp().Logger().
		With().Caller().Logger().
		With().Str("role", "cli node").Logger()

}

func main() {
	app := &urfave.App{
		Name:  "Node controller",
		Usage: "Please use the start command",

		Commands: []*urfave.Command{
			{
				Name:  "start",
				Usage: "starts the node and proxy",
				Flags: []urfave.Flag{
					&urfave.StringFlag{
						Name:  "proxyaddr",
						Usage: "addr of the proxy",
						Value: "127.0.0.1:0",
					},
					&urfave.StringFlag{
						Name:  "nodeaddr",
						Usage: "addr of the node",
						Value: "127.0.0.1:0",
					},
					&urfave.DurationFlag{
						Name:  "antientropy",
						Usage: "Antientropy interval",
						// 0 means the antientropy is not activated
						Value: time.Second * 10, //0, now we need 10s
					},
					&urfave.DurationFlag{
						Name:  "heartbeat",
						Usage: "Heartbeat interval",
						// 0 means the heartbeat is not activated
						Value: 0,
					},
					&urfave.DurationFlag{
						Name:  "acktimeout",
						Usage: "Timeout of ack message",
						// this is considered as a reasonable timeout value for
						// a small system.
						Value: time.Second * 10, //time.Second * 3, //in project we need 10s
					},
					&urfave.Float64Flag{
						Name:  "continuemongering",
						Usage: "probability to continue mongering",
						// by default there is a 50% chance to continue
						// mongering.
						Value: 0.5,
					},
					&urfave.StringFlag{
						Name:  "storagefolder",
						Usage: "folder that will store peer's data. If not set will use in-memory storage",
						Value: "",
					},
					&urfave.UintFlag{
						Name:  "chunksize",
						Usage: "Size of chunks, in bytes",
						Value: 8192,
					},
					&urfave.DurationFlag{
						Name:  "backoffinitial",
						Usage: "Initial time for the backoff strategy",
						Value: time.Second * 2,
					},
					&urfave.UintFlag{
						Name:  "backofffactor",
						Usage: "Factor value for the backoff strategy",
						Value: 2,
					},
					&urfave.UintFlag{
						Name:  "backoffretry",
						Usage: "Retry value for the backoff strategy",
						Value: 5,
					},
					&urfave.UintFlag{
						Name:  "totalpeers",
						Usage: "Total number of peers (needed for Paxos)",
						Value: 3, // for project we modify it to 3 instead of 1
					},
					&urfave.UintFlag{
						Name:  "paxosid",
						Usage: "The peer's paxos id. Must stat at 1. Can be 0 if total peers <= 1.",
						Value: 0,
					},
					&urfave.DurationFlag{
						Name:  "paxosproposerretry",
						Usage: "The timeout after which a paxos proposer retries",
						Value: time.Second * 5,
					},
					//---------project start
					&urfave.StringFlag{
						Name:  "configpath",
						Usage: "The path to a json file that determines the behaviors of the peer",
						Value: "./config_0.json",
					},
					&urfave.BoolFlag{
						Name:  "pbft",
						Usage: "The flag to decide if we would apply pbft(for visualization)",
						Value: true,
					},
					// ----- project ends
				},
				Action: start,
			},
		},

		Action: func(c *urfave.Context) error {
			urfave.ShowAppHelpAndExit(c, 1)
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
}

// ----- project starts
type ProjectConfig struct {
	// every node MAY have multiple roles
	Role []string

	// for Cothority Node
	Index         uint
	Addresses     []string
	HRSecretTable map[uint]string // It is used for HMAC check file request

	// for HR
	ID     uint
	Secret string
	Num    uint

	// for Applicant
	// Nothing
}

func ReadJSON(configPath string) ProjectConfig {
	filePtr, err := os.Open(configPath)
	if err != nil {
		return ProjectConfig{}
	}
	defer filePtr.Close()
	var info ProjectConfig
	decoder := json.NewDecoder(filePtr)
	err = decoder.Decode(&info)
	if err != nil {
		fmt.Println("decode fails", err.Error())
	} else {
		fmt.Println("decode sucess")
		fmt.Println(info)
	}
	return info
}

// ----- project ends

// start starts the http proxy. It will create a UDP socket.
func start(c *urfave.Context) error {
	// ----- project starts
	projectconfig := ReadJSON(c.String("configpath"))
	// ----- project ends

	proxyAddr := c.String("proxyaddr")
	var nodeAddr string
	if len(projectconfig.Addresses) > 0 {
		nodeAddr = projectconfig.Addresses[projectconfig.Index]
	} else {
		nodeAddr = c.String("nodeaddr")
	}

	trans := udp.NewUDP()

	sock, err := trans.CreateSocket(nodeAddr)
	if err != nil {
		return xerrors.Errorf("failed to create socket")
	}

	// this message is used by the binary node to get the peer address
	log.Info().Msgf(peerAddrMsg, sock.GetAddress())

	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("socketaddress_%d", os.Getpid()))

	err = os.WriteFile(socketPath, []byte(sock.GetAddress()), os.ModePerm)
	if err != nil {
		return xerrors.Errorf("failed to write socket address file: %v", err)
	}

	var storage storage.Storage

	if c.String("storagefolder") == "" {
		storage = inmemory.NewPersistency()
	} else {
		storage, err = file.NewPersistency(c.String("storagefolder"))
		if err != nil {
			log.Fatal().Msgf("failed to create file storage: %v", err)
		}
	}
	totalPeers := c.Uint("totalpeers")
	//paxosID := c.Uint("paxosid")
	// in our project, we need to assign new id
	paxosID := projectconfig.Index + 1

	if totalPeers > 1 && paxosID == 0 {
		return xerrors.Errorf("if total peers is set PaxosID must be set, too")
	}

	conf := peer.Configuration{
		Socket:          sock,
		MessageRegistry: standard.NewRegistry(),

		AntiEntropyInterval: c.Duration("antientropy"),
		HeartbeatInterval:   c.Duration("heartbeat"),
		AckTimeout:          c.Duration("acktimeout"),
		ContinueMongering:   c.Float64("continuemongering"),

		ChunkSize: c.Uint("chunksize"),
		BackoffDataRequest: peer.Backoff{
			Initial: c.Duration("backoffinitial"),
			Factor:  c.Uint("backofffactor"),
			Retry:   c.Uint("backoffretry"),
		},
		Storage: storage,
		//for project, we need to initialize storage
		HrStorage: inmemory.NewHrPersistency(),

		TotalPeers: totalPeers,
		PaxosThreshold: func(u uint) int {
			return int(u/2 + 1)
		},
		PaxosID:            paxosID,
		PaxosProposerRetry: c.Duration("paxosproposerretry"),
		CothorityAddress:   projectconfig.Addresses,
		UsePBFT:            c.Bool("pbft"),
	}
	node := peerFactory(conf)

	role := projectconfig.Role
	if len(role) > 0 {
		node.AddPeer(projectconfig.Addresses...)
		if role[0] == "CN" {
			node.CreateCothorityNode(projectconfig.Index, projectconfig.Addresses, projectconfig.HRSecretTable)
			// if role[1] == "HR" {
			// 	node.CreateHR(projectconfig.ID, projectconfig.Secret, projectconfig.Num)
			// } else if role[1] == "AP" {
			// 	node.CreateApplicant()
			// }
			node.StartCothorityNode()
			node.CreateHR(projectconfig.ID, projectconfig.Secret, projectconfig.Num)
			node.CreateApplicant()
		}
	}
	httpnode := httpnode.NewHTTPNode(node, conf)

	notify := make(chan os.Signal, 1)
	signal.Notify(notify,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	err = httpnode.StartAndListen(proxyAddr)
	if err != nil {
		return xerrors.Errorf("failed to start and listen: %v", err)
	}

	<-notify
	log.Info().Msg("closing...")

	os.RemoveAll(socketPath)

	err = httpnode.StopAndClose()
	if err != nil {
		return xerrors.Errorf("failed to close: %v", err)
	}

	sock.Close()

	return nil
}
