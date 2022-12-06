package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func createTableWriter() table.Writer {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	style := table.StyleDefault
	style.Format.Header = text.FormatDefault
	t.SetStyle(style)
	return t
}

type Series struct {
	Name    string          `json:"name"`
	Columns []string        `json:"columns"`
	Values  [][]interface{} `json:"values"`
}

type InfluxQueryResult struct {
	Results []struct {
		StatementID int      `json:"statement_id"`
		Series      []Series `json:"series"`
	} `json:"results"`
}

func influxTablePrint(ugly bool, influxDB bool, dat []byte) bool {
	if ugly || !influxDB {
		return false
	}

	var qr InfluxQueryResult
	_ = json.Unmarshal(dat, &qr)
	if len(qr.Results) > 0 && len(qr.Results[0].Series) > 0 {
		for _, series := range qr.Results[0].Series {
			influxSeriesPrint(series)
		}
		return true
	}

	return false
}

func influxSeriesPrint(series Series) {
	fmt.Printf("%s:\n", series.Name)
	header := make([]any, 1+len(series.Columns))
	header[0] = "#"
	for i, h := range series.Columns {
		header[i+1] = h
	}
	tw := createTableWriter()
	tw.AppendHeader(header)

	for i, cells := range series.Values {
		tw.AppendRow(append([]any{fmt.Sprintf("%d", i+1)}, cells...))
	}
	tw.Render()
}
