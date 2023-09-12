package block

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"
)

func LastBlockCmd() *cobra.Command {
	var height string
	cmd := &cobra.Command{
		Use:   "block",
		Short: "Get a specific block persisted in the db. If height is not specified, defaults to the latest.",
		Long:  "Get the last block persisted in the db. If height is not specified, defaults to the latest.\nThis command works only if no other process is using the db. Before using it, make sure to stop your node.\nIf you're using a custom home directory, specify it with the '--home' flag",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)

			// Bind flags to the Context's Viper so the app construction can set
			// options accordingly.
			err := serverCtx.Viper.BindPFlags(cmd.Flags())
			if err != nil {
				return err
			}

			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)
			cfg := serverCtx.Config
			home := cfg.RootDir

			statedb, err := newStateStore(home, server.GetAppDBBackend(serverCtx.Viper))
			if err != nil {
				return fmt.Errorf("error while openning db: %w", err)
			}

			blockStore := statedb.loadBlockStoreState()
			if blockStore == nil {
				return errors.New("couldn't find a BlockStoreState persisted in db")
			}

			var reqHeight int64
			if height != "latest" {
				reqHeight, err = strconv.ParseInt(height, 10, 64)
				if err != nil {
					return errors.New("invalid height, please provide an integer")
				}
				if reqHeight > blockStore.Height {
					return fmt.Errorf("invalid height, the latest height found in the db is %d, and you asked for %d", blockStore.Height, reqHeight)
				}
			} else {
				reqHeight = blockStore.Height
			}

			block := statedb.loadBlock(reqHeight)

			bz, err := json.Marshal(block)
			if err != nil {
				return err
			}

			fmt.Println(string(bz))

			return nil
		},
	}

	cmd.Flags().StringVar(&height, "height", "latest", "Block height to retrieve")
	return cmd
}