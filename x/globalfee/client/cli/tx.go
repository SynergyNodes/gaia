package cli

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gaia/v9/x/globalfee/types"
)

func GetTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Tx commands for the global fee module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	txCmd.AddCommand(
		NewMultiSendCmd(),
	)
	return txCmd
}

func parseCSV(path string) [][]string {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		panic(err)
	}

	return records
}

// NewMultiSendCmd returns a CLI command for multi-send
func NewMultiSendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "multi-send [csv_file] [denom] [startIndex] [threshold]",
		Short:   "Execute multisend based on csv file",
		Example: `gaiad tx globalfee multi-send "./airdrop.csv" acnto 0 100 --from=WALLET --keyring-backend=test`,
		Args:    cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			sendMsgs := []banktypes.MsgSend{}
			amountRecords := parseCSV(args[0])

			for _, line := range amountRecords[1:] {
				addr, amountStr := line[0], line[1]
				amountDec := sdk.MustNewDecFromStr(amountStr)
				decimalReduction := sdk.NewInt(1000_000_000).Mul(sdk.NewInt(1000_000_000)) // 10^18
				amount := amountDec.Mul(decimalReduction.ToDec()).TruncateInt()

				msg := banktypes.MsgSend{
					FromAddress: clientCtx.FromAddress.String(),
					ToAddress:   addr,
					Amount:      sdk.Coins{sdk.NewCoin(args[1], amount)},
				}
				sendMsgs = append(sendMsgs, msg)
			}

			startIndex, err := strconv.Atoi(args[2])
			if err != nil {
				return err
			}
			threshold, err := strconv.Atoi(args[3])
			if err != nil {
				return err
			}

			msgs := []sdk.Msg{}
			for index, msg := range sendMsgs {
				if index < startIndex {
					continue
				}
				msgs = append(msgs, &banktypes.MsgSend{
					FromAddress: msg.FromAddress,
					ToAddress:   msg.ToAddress,
					Amount:      msg.Amount,
				})
				if len(msgs) >= threshold || index+1 == len(sendMsgs) {
					err := tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msgs...)
					if err != nil {
						return err
					}
					fmt.Printf("executed batch %d/%d\n", index+1, len(sendMsgs))
					msgs = []sdk.Msg{}
				}
			}
			fmt.Println("finalized batch execution")

			return nil
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
