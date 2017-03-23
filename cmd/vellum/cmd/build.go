//  Copyright (c) 2017 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 		http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"

	"strconv"

	"github.com/couchbaselabs/vellum"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Builds a vellum FST file from a CSV file.",
	Long:  `Builds a vellum FST file from a CSV file.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("paths required")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		inFile, err := os.Open(args[0])
		if err != nil {
			return err
		}

		outFile, err := os.Create(args[1])
		if err != nil {
			return err
		}
		builder, err := vellum.New(outFile, nil)
		if err != nil {
			return err
		}

		r := csv.NewReader(inFile)
		recordCount := 0
		for {
			record, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			if len(record) != 2 {
				return fmt.Errorf("incorrect number of records: %v", record)
			}
			val, err := strconv.ParseUint(record[1], 10, 64)
			if err != nil {
				return err
			}
			err = builder.Insert([]byte(record[0]), val)
			if err != nil {
				return err
			}
			recordCount += 1
		}
		builder.Close()
		fmt.Printf("inserted %v records\n", recordCount)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(buildCmd)
}
