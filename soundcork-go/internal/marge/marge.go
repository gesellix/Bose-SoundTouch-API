package marge

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/deborahgu/soundcork/internal/constants"
	"github.com/deborahgu/soundcork/internal/datastore"
	"github.com/deborahgu/soundcork/internal/models"
)

const DateStr = "2012-09-19T12:43:00.000+00:00"

func SourceProviders() []models.SourceProvider {
	providers := make([]models.SourceProvider, len(constants.Providers))
	for i, name := range constants.Providers {
		providers[i] = models.SourceProvider{
			ID:        i + 1,
			CreatedOn: DateStr,
			Name:      name,
			UpdatedOn: DateStr,
		}
	}
	return providers
}

type SourceProvidersXML struct {
	XMLName   xml.Name                `xml:"sourceProviders"`
	Providers []models.SourceProvider `xml:"sourceProvider"`
}

func SourceProvidersToXML() ([]byte, error) {
	sp := SourceProvidersXML{
		Providers: SourceProviders(),
	}
	data, err := xml.MarshalIndent(sp, "", "    ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), data...), nil
}

func ConfiguredSourceToXML(cs models.ConfiguredSource) ([]byte, error) {
	type SourceXML struct {
		XMLName    xml.Name `xml:"source"`
		ID         string   `xml:"id,attr"`
		Type       string   `xml:"type,attr"`
		CreatedOn  string   `xml:"createdOn"`
		Credential struct {
			Type  string `xml:"type,attr"`
			Value string `xml:",chardata"`
		} `xml:"credential"`
		Name             string `xml:"name"`
		SourceProviderID string `xml:"sourceproviderid"`
		SourceName       string `xml:"sourcename"`
		SourceSettings   string `xml:"sourcesettings"`
		UpdatedOn        string `xml:"updatedOn"`
		Username         string `xml:"username"`
	}

	providerID := 0
	for i, p := range constants.Providers {
		if p == cs.SourceKeyType {
			providerID = i + 1
			break
		}
	}

	sxml := SourceXML{
		ID:               cs.ID,
		Type:             "Audio",
		CreatedOn:        DateStr,
		Name:             cs.SourceKeyAccount,
		SourceProviderID: strconv.Itoa(providerID),
		SourceName:       cs.DisplayName,
		UpdatedOn:        DateStr,
		Username:         cs.SourceKeyAccount,
	}
	sxml.Credential.Type = "token"
	sxml.Credential.Value = cs.Secret

	return xml.Marshal(sxml)
}

func GetConfiguredSourceXML(cs models.ConfiguredSource) string {
	providerID := 0
	for i, p := range constants.Providers {
		if p == cs.SourceKeyType {
			providerID = i + 1
			break
		}
	}
	return fmt.Sprintf(`<source id="%s" type="Audio"><createdOn>%s</createdOn><credential type="token">%s</credential><name>%s</name><sourceproviderid>%d</sourceproviderid><sourcename>%s</sourcename><sourcesettings></sourcesettings><updatedOn>%s</updatedOn><username>%s</username></source>`,
		cs.ID, DateStr, cs.Secret, cs.SourceKeyAccount, providerID, cs.DisplayName, DateStr, cs.SourceKeyAccount)
}

func PresetsToXML(ds *datastore.DataStore, account string) ([]byte, error) {
	presets, err := ds.GetPresets(account)
	if err != nil {
		return nil, err
	}
	sources, err := ds.GetConfiguredSources(account)
	if err != nil {
		return nil, err
	}

	res := `<presets>`
	for _, p := range presets {
		res += fmt.Sprintf(`<preset buttonNumber="%s">`, p.ID)
		res += fmt.Sprintf(`<containerArt>%s</containerArt>`, p.ContainerArt)
		res += fmt.Sprintf(`<contentItemType>%s</contentItemType>`, p.Type)
		res += fmt.Sprintf(`<createdOn>%s</createdOn>`, DateStr)
		res += fmt.Sprintf(`<location>%s</location>`, p.Location)
		res += fmt.Sprintf(`<name>%s</name>`, p.Name)

		// Content Item Source
		found := false
		for _, s := range sources {
			if s.ID == p.SourceID || (s.SourceKeyType == p.Source && s.SourceKeyAccount == p.SourceAccount) {
				res += GetConfiguredSourceXML(s)
				found = true
				break
			}
		}
		if !found {
			// This might happen if source is not found
		}

		res += fmt.Sprintf(`<updatedOn>%s</updatedOn>`, DateStr)
		res += `</preset>`
	}
	res += `</presets>`

	return append([]byte(xml.Header), []byte(res)...), nil
}

func RecentsToXML(ds *datastore.DataStore, account string) ([]byte, error) {
	recents, err := ds.GetRecents(account)
	if err != nil {
		return nil, err
	}
	sources, err := ds.GetConfiguredSources(account)
	if err != nil {
		return nil, err
	}

	res := `<recents>`
	for _, r := range recents {
		lastPlayed := ""
		if sec, err := strconv.ParseInt(r.UtcTime, 10, 64); err == nil {
			lastPlayed = time.Unix(sec, 0).Format(time.RFC3339)
		}

		res += fmt.Sprintf(`<recent id="%s">`, r.ID)
		res += fmt.Sprintf(`<contentItemType>%s</contentItemType>`, r.Type)
		res += fmt.Sprintf(`<createdOn>%s</createdOn>`, DateStr)
		res += fmt.Sprintf(`<lastplayedat>%s</lastplayedat>`, lastPlayed)
		res += fmt.Sprintf(`<location>%s</location>`, r.Location)
		res += fmt.Sprintf(`<name>%s</name>`, r.Name)

		found := false
		for _, s := range sources {
			if s.ID == r.SourceID || (s.SourceKeyType == r.Source && s.SourceKeyAccount == r.SourceAccount) {
				res += GetConfiguredSourceXML(s)
				found = true
				break
			}
		}
		if !found {
		}

		res += fmt.Sprintf(`<updatedOn>%s</updatedOn>`, DateStr)
		res += `</recent>`
	}
	res += `</recents>`

	return append([]byte(xml.Header), []byte(res)...), nil
}

func ProviderSettingsToXML(account string) string {
	return fmt.Sprintf(`<providerSettings><providerSetting><boseId>%s</boseId><keyName>ELIGIBLE_FOR_TRIAL</keyName><value>true</value><providerId>14</providerId></providerSetting></providerSettings>`, account)
}

func SoftwareUpdateToXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><software_update><softwareUpdateLocation></softwareUpdateLocation></software_update>`
}

func AccountFullToXML(ds *datastore.DataStore, account string) ([]byte, error) {
	devicesDir := ds.AccountDevicesDir(account)
	entries, err := os.ReadDir(devicesDir)
	if err != nil {
		return nil, err
	}

	res := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><account id="%s"><accountStatus>OK</accountStatus><devices>`, account)
	lastDeviceID := ""
	for _, entry := range entries {
		if entry.IsDir() {
			deviceID := entry.Name()
			lastDeviceID = deviceID
			info, err := ds.GetDeviceInfo(account, deviceID)
			if err != nil {
				continue
			}

			res += fmt.Sprintf(`<device deviceid="%s">`, deviceID)
			res += fmt.Sprintf(`<attachedProduct product_code="%s"><components/><productlabel>%s</productlabel><serialnumber>%s</serialnumber></attachedProduct>`,
				info.ProductCode, info.ProductCode, info.ProductSerialNumber)
			res += fmt.Sprintf(`<createdOn>%s</createdOn>`, DateStr)
			res += fmt.Sprintf(`<firmwareVersion>%s</firmwareVersion>`, info.FirmwareVersion)
			res += fmt.Sprintf(`<ipaddress>%s</ipaddress>`, info.IPAddress)
			res += fmt.Sprintf(`<name>%s</name>`, info.Name)

			presets, _ := PresetsToXML(ds, account)
			res += string(presets[len(xml.Header):]) // strip header

			recents, _ := RecentsToXML(ds, account)
			res += string(recents[len(xml.Header):]) // strip header

			res += fmt.Sprintf(`<serialnumber>%s</serialnumber>`, info.DeviceSerialNumber)
			res += fmt.Sprintf(`<updatedOn>%s</updatedOn>`, DateStr)
			res += `</device>`
		}
	}
	res += `</devices><mode>global</mode><preferrendLanguage>en</preferrendLanguage>`
	res += ProviderSettingsToXML(account)

	if lastDeviceID != "" {
		sources, _ := ds.GetConfiguredSources(account)
		res += `<sources>`
		for _, s := range sources {
			res += GetConfiguredSourceXML(s)
		}
		res += `</sources>`
	}

	res += `</account>`
	return []byte(res), nil
}
