package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/ribtoks/listing/pkg/common"
	"gopkg.in/yaml.v2"
)

type Printer interface {
	Append(s *common.Subscriber)
	Render() error
}

func structToMap(cs *common.SubscriberEx) map[string]string {
	values := make(map[string]string)
	s := reflect.ValueOf(cs).Elem()
	typeOfT := s.Type()

	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		var v string
		switch f.Interface().(type) {
		case bool:
			v = fmt.Sprintf("%v", f.Bool())
		case int, int8, int16, int32, int64:
			v = strconv.FormatInt(f.Int(), 10)
		case uint, uint8, uint16, uint32, uint64:
			v = strconv.FormatUint(f.Uint(), 10)
		case float32:
			v = strconv.FormatFloat(f.Float(), 'f', 4, 32)
		case float64:
			v = strconv.FormatFloat(f.Float(), 'f', 4, 64)
		case []byte:
			v = string(f.Bytes())
		case string:
			v = f.String()
		case common.JSONTime:
			v = f.Interface().(common.JSONTime).String()
			v = strings.Trim(v, `"`)
		}
		values[typeOfT.Field(i).Name] = v
	}
	return values
}

func mapValues(m map[string]string, fields []string) []string {
	row := make([]string, 0, len(fields))
	for _, k := range fields {
		if v, ok := m[k]; ok {
			row = append(row, v)
		}
	}
	return row
}

type TablePrinter struct {
	secret string
	table  *tablewriter.Table
	fields []string
}

func SubscriberHeaders() []string {
	statType := reflect.TypeOf(common.SubscriberEx{})
	header := make([]string, 0, statType.NumField())
	for i := 0; i < statType.NumField(); i++ {
		field := statType.Field(i)
		header = append(header, field.Name)
	}
	return header
}

func NewTablePrinter(secret string) *TablePrinter {
	tr := &TablePrinter{
		secret: secret,
		table:  tablewriter.NewWriter(os.Stdout),
		fields: SubscriberHeaders(),
	}

	tr.table.SetHeader(tr.fields)
	return tr
}

func NewTSVPrinter(secret string) *TablePrinter {
	tr := NewTablePrinter(secret)

	tr.table.SetAutoWrapText(false)
	tr.table.SetAutoFormatHeaders(true)
	tr.table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	tr.table.SetAlignment(tablewriter.ALIGN_LEFT)
	tr.table.SetCenterSeparator("")
	tr.table.SetColumnSeparator("")
	tr.table.SetRowSeparator("")
	tr.table.SetHeaderLine(false)
	tr.table.SetBorder(false)
	tr.table.SetTablePadding("\t") // pad with tabs
	tr.table.SetNoWhiteSpace(true)

	return tr
}

func (tr *TablePrinter) Append(s *common.Subscriber) {
	se := common.NewSubscriberEx(s, tr.secret)
	m := structToMap(se)
	row := mapValues(m, tr.fields)
	tr.table.Append(row)
}

func (tr *TablePrinter) Render() error {
	tr.table.Render()
	return nil
}

type CSVPrinter struct {
	secret string
	w      *csv.Writer
	fields []string
}

func NewCSVPrinter(secret string) *CSVPrinter {
	cr := &CSVPrinter{
		secret: secret,
		w:      csv.NewWriter(os.Stdout),
		fields: SubscriberHeaders(),
	}
	cr.w.Write(cr.fields)
	return cr
}

func (cr *CSVPrinter) Append(s *common.Subscriber) {
	se := common.NewSubscriberEx(s, cr.secret)
	m := structToMap(se)
	row := mapValues(m, cr.fields)
	cr.w.Write(row)
}

func (cr *CSVPrinter) Render() error {
	cr.w.Flush()
	return nil
}

type JsonPrinter struct {
	subscribers []*common.SubscriberEx
	secret      string
}

func NewJsonPrinter(secret string) *JsonPrinter {
	rp := &JsonPrinter{
		subscribers: make([]*common.SubscriberEx, 0),
		secret:      secret,
	}
	return rp
}

func (rp *JsonPrinter) Append(s *common.Subscriber) {
	rp.subscribers = append(rp.subscribers, common.NewSubscriberEx(s, rp.secret))
}

func (rp *JsonPrinter) Render() error {
	data, err := json.MarshalIndent(rp.subscribers, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

type RawPrinter struct {
	subscribers []*common.Subscriber
}

func NewRawPrinter() *RawPrinter {
	rp := &RawPrinter{
		subscribers: make([]*common.Subscriber, 0),
	}
	return rp
}

func (rp *RawPrinter) Append(s *common.Subscriber) {
	rp.subscribers = append(rp.subscribers, s)
}

func (rp *RawPrinter) Render() error {
	data, err := json.MarshalIndent(rp.subscribers, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

type YamlPrinter struct {
	secret      string
	subscribers []*common.SubscriberEx
}

func NewYamlPrinter(secret string) *YamlPrinter {
	yp := &YamlPrinter{
		secret:      secret,
		subscribers: make([]*common.SubscriberEx, 0),
	}
	return yp
}

func (yp *YamlPrinter) Append(s *common.Subscriber) {
	se := common.NewSubscriberEx(s, yp.secret)
	yp.subscribers = append(yp.subscribers, se)
}

func (yp *YamlPrinter) Render() error {
	data, err := yaml.Marshal(yp.subscribers)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
