package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/icon-project/goloop/chain"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/network"
)

type GoChainConfig struct {
	chain.Config
	P2PAddr string `json:"p2p"`
	Key     []byte `json:"key"`

	fileName string
}

func (config *GoChainConfig) String() string {
	return ""
}

func (config *GoChainConfig) Set(name string) error {
	config.fileName = name
	if bs, e := ioutil.ReadFile(name); e == nil {
		if err := json.Unmarshal(bs, config); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	var configFile, genesisFile string
	var generate bool
	var cfg GoChainConfig

	flag.Var(&cfg, "config", "Parsing configuration file")
	flag.BoolVar(&generate, "gen", false, "Generate configuration file")
	flag.StringVar(&cfg.Channel, "channel", "default", "Channel name for the chain")
	flag.StringVar(&cfg.P2PAddr, "p2p", "127.0.0.1:8080", "Listen ip-port of P2P")
	flag.IntVar(&cfg.NID, "nid", 1, "Chain Network ID")
	flag.StringVar(&cfg.RPCAddr, "rpc", ":9080", "Listen ip-port of JSON-RPC")
	flag.StringVar(&cfg.SeedAddr, "seed", "", "Ip-port of Seed")
	flag.StringVar(&genesisFile, "genesis", "", "Genesis transaction param")
	flag.StringVar(&cfg.DBType, "db_type", "mapdb", "Name of database system(*badgerdb, goleveldb, boltdb, mapdb)")
	flag.StringVar(&cfg.DBDir, "db_dir", "", "Database directory")
	flag.StringVar(&cfg.DBName, "db_name", "", "Database name for the chain(default:<channel name>)")
	flag.UintVar(&cfg.Role, "role", 0, "[0:None, 1:Seed, 2:Validator, 3:Both]")
	flag.Parse()

	if len(genesisFile) > 0 {
		genesis, err := ioutil.ReadFile(genesisFile)
		if err != nil {
			log.Panicf("Fail to open genesis file=%s err=%+v", genesisFile, err)
		}
		cfg.Genesis = genesis
	}

	key := cfg.Key
	var priK *crypto.PrivateKey
	if len(key) == 0 {
		priK, _ = crypto.GenerateKeyPair()
		cfg.Key = priK.Bytes()
	} else {
		var err error
		if priK, err = crypto.ParsePrivateKey(key); err != nil {
			log.Panicf("Illegal key data=[%x]", key)
		}
	}

	if cfg.DBDir == "" {
		addr := common.NewAccountAddressFromPublicKey(priK.PublicKey())
		cfg.DBDir = ".db/" + addr.String()
	}

	if generate {
		if len(cfg.fileName) == 0 {
			cfg.fileName = "config.json"
		}
		f, err := os.OpenFile(cfg.fileName,
			os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			log.Panicf("Fail to open file=%s err=%+v", configFile, err)
		}

		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(&cfg); err != nil {
			log.Panicf("Fail to generate JSON for %+v", cfg)
		}
		f.Close()
		os.Exit(0)
	}

	if cfg.DBType != "mapdb" {
		if err := os.MkdirAll(cfg.DBDir, 0755); err != nil {
			log.Panicf("Fail to create directory %s err=%+v", cfg.DBDir, err)
		}
	}

	wallet, _ := common.WalletFromPrivateKey(priK)
	nt := network.NewTransport(cfg.P2PAddr, wallet)
	nt.Listen()
	defer nt.Close()

	c := chain.NewChain(wallet, nt, &cfg.Config)
	c.Start()
}
