package zenon

import (
	"path"

	"github.com/zenon-network/go-zenon/chain/store"
	"github.com/zenon-network/go-zenon/common/db"
	"github.com/zenon-network/go-zenon/wallet"
)

type Config struct {
	MinPeers         int
	DataDir          string
	ProducingKeyPair *wallet.KeyPair
	GenesisConfig    store.Genesis
}

func (c *Config) NewDBManager(inside string) db.Manager {
	return db.NewLevelDBManager(path.Join(c.DataDir, inside))
}
func (c *Config) NewLevelDB(inside string) db.DB {
	return db.NewLevelDB(path.Join(c.DataDir, inside))
}
