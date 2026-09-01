package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"heimdall/internal/agent_adapters/antigravity/forensics"
)

const (
	executorAuditTabMinimumWidth = 0
	executorAuditTabWidth        = 4
	executorAuditTabPadding      = 1
	executorAuditTabPadCharacter = ' '
	executorAuditTabFlags        = 0
	executorAuditDefaultIndex    = -1
	executorAuditMissingValue    = "-"
)

func main() {
	databasePath := flag.String("db", "", "read-only path to an Antigravity conversation SQLite database")
	executorIndex := flag.Int("idx", executorAuditDefaultIndex, "executor_metadata index for printable-string detail")
	flag.Parse()

	audit, auditErr := forensics.ReadExecutorMetadataAudit(*databasePath)
	if auditErr != nil {
		fmt.Fprintln(os.Stderr, auditErr)
		os.Exit(1)
	}
	if *executorIndex == executorAuditDefaultIndex {
		if err := writeExecutorMetadataAudit(os.Stdout, audit); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	row, found := findExecutorMetadataAuditRow(audit.Rows, *executorIndex)
	if !found {
		fmt.Fprintf(os.Stderr, "executor_metadata idx %d was not found\n", *executorIndex)
		os.Exit(1)
	}
	printableStrings, stringsErr := forensics.ReadExecutorMetadataPrintableStrings(*databasePath, *executorIndex)
	if stringsErr != nil {
		fmt.Fprintln(os.Stderr, stringsErr)
		os.Exit(1)
	}
	if err := writeExecutorMetadataDetail(os.Stdout, row, printableStrings); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func writeExecutorMetadataAudit(output io.Writer, audit forensics.ExecutorMetadataAudit) error {
	writer := tabwriter.NewWriter(output, executorAuditTabMinimumWidth, executorAuditTabWidth, executorAuditTabPadding, executorAuditTabPadCharacter, executorAuditTabFlags)
	if _, err := fmt.Fprintln(writer, "EXECUTOR IDX\tOBSERVED UUIDS\tMODEL\tBYTES"); err != nil {
		return err
	}
	for _, row := range audit.Rows {
		if _, err := fmt.Fprintf(writer, "%d\t%s\t%s\t%d\n", row.Index, executorAuditUUIDSummary(row.ObservedExecutionUUIDs), executorAuditDisplayValue(row.ModelName), row.ByteCount); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(writer, "TOTAL\t%d executor metadata rows\n", len(audit.Rows)); err != nil {
		return err
	}
	return writer.Flush()
}

func writeExecutorMetadataDetail(output io.Writer, row forensics.ExecutorMetadataAuditRow, printableStrings []string) error {
	if _, err := fmt.Fprintf(output, "EXECUTOR METADATA IDX %d\n", row.Index); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "BYTES: %d\nMODEL: %s\nOBSERVED UUIDS: %s\n", row.ByteCount, executorAuditDisplayValue(row.ModelName), executorAuditUUIDSummary(row.ObservedExecutionUUIDs)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(output, "OBSERVED PRINTABLE STRINGS (schema-inferred; not a protobuf field listing):"); err != nil {
		return err
	}
	for _, value := range printableStrings {
		if _, err := fmt.Fprintf(output, "- %s\n", value); err != nil {
			return err
		}
	}
	return nil
}

func findExecutorMetadataAuditRow(rows []forensics.ExecutorMetadataAuditRow, targetIndex int) (forensics.ExecutorMetadataAuditRow, bool) {
	for _, row := range rows {
		if row.Index == targetIndex {
			return row, true
		}
	}
	return forensics.ExecutorMetadataAuditRow{}, false
}

func executorAuditDisplayValue(value string) string {
	if value == "" {
		return executorAuditMissingValue
	}
	return value
}

func executorAuditUUIDSummary(values []string) string {
	if len(values) == 0 {
		return executorAuditMissingValue
	}
	return strings.Join(values, ",")
}
