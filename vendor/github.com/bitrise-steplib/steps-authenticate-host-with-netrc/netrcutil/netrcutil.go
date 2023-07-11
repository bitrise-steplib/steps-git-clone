package netrcutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

const netrcDefaultFileName = ".netrc"

// NetRCItemModel ...
type NetRCItemModel struct {
	Machine  string
	Login    string
	Password string
}

// NetRCModel ...
type NetRCModel struct {
	OutputPth  string
	ItemModels []NetRCItemModel
}

// New ...
func New() *NetRCModel {
	netRCPth := filepath.Join(pathutil.UserHomeDir(), netrcDefaultFileName)
	return &NetRCModel{OutputPth: netRCPth}
}

func (netRCModel *NetRCModel) CreateOrUpdateFile(itemModels ...NetRCItemModel) error {
	netRCModel.AddItemModel(itemModels...)

	log.Infof("Writing .netrc file...")

	isExists, err := pathutil.IsPathExists(netRCModel.OutputPth)
	if err != nil {
		return fmt.Errorf("Failed to check path (%s), error: %s", netRCModel.OutputPth, err)
	}

	if !isExists {
		log.Printf("No .netrc file found at (%s), creating new...", netRCModel.OutputPth)

		if err := netRCModel.CreateFile(); err != nil {
			return fmt.Errorf("Failed to write .netrc file, error: %s", err)
		}
	} else {
		log.Warnf("File already exists at (%s)", netRCModel.OutputPth)

		backupPth := fmt.Sprintf("%s%s", strings.Replace(netRCModel.OutputPth, ".netrc", ".bk.netrc", -1), time.Now().Format("2006_01_02_15_04_05"))

		if originalContent, err := fileutil.ReadBytesFromFile(netRCModel.OutputPth); err != nil {
			return fmt.Errorf("Failed to read file (%s), error: %s", netRCModel.OutputPth, err)
		} else if err := fileutil.WriteBytesToFile(backupPth, originalContent); err != nil {
			return fmt.Errorf("Failed to write file (%s), error: %s", backupPth, err)
		} else {
			log.Printf("Backup created at: %s", backupPth)
		}

		log.Printf("Appending config to the existing .netrc file...")

		if err := netRCModel.Append(); err != nil {
			return fmt.Errorf("Failed to write .netrc file, error: %s", err)
		}
	}

	return nil
}

// AddItemModel ...
func (netRCModel *NetRCModel) AddItemModel(itemModels ...NetRCItemModel) {
	netRCModel.ItemModels = append(netRCModel.ItemModels, itemModels...)
}

// CreateFile ...
func (netRCModel *NetRCModel) CreateFile() error {
	netRCFileContent := generateFileContent(netRCModel)
	permission := os.FileMode(0600) // Other tools might fail if the file's permission is not 0600
	return fileutil.WriteStringToFileWithPermission(netRCModel.OutputPth, netRCFileContent, permission)
}

// Append ...
func (netRCModel *NetRCModel) Append() error {
	netRCFileContent := generateFileContent(netRCModel)
	return fileutil.AppendStringToFile(netRCModel.OutputPth, fmt.Sprintf("\n\n%s", netRCFileContent))
}

func generateFileContent(netRCModel *NetRCModel) string {
	netRCFileContent := ""
	for i, itemModel := range netRCModel.ItemModels {
		netRCFileContent += fmt.Sprintf("machine %s\n\tlogin %s\n\tpassword %s\n", itemModel.Machine, itemModel.Login, itemModel.Password)
		if i != len(netRCModel.ItemModels)-1 {
			netRCFileContent += "\n\n"
		}
	}
	return netRCFileContent
}
