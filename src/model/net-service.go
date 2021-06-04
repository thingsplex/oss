package model

import (
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
)

type NetworkService struct {
	mqt *fimpgo.MqttTransport
}

// Inspiration taken from github.com/tskaard/fh-tibber
func createInterface(iType string, msgType string, valueType string, version string) fimptype.Interface {
	inter := fimptype.Interface{
		Type:      iType,
		MsgType:   msgType,
		ValueType: valueType,
		Version:   version,
	}
	return inter
}

func createMeterService(addr string, service string, alias string) fimptype.Service {
	props := make(map[string]interface{})
	props["sup_units"] = []string{"W"}
	props["sup_extended_vals"] = []string{
		"p_import", "e_import", "e_export",
		"last_e_import", "last_e_export",
		"p_import_min", "p_import_avg", "p_import_max",
		"p_export", "p_export_min", "p_export_max",
		"u1", "u2", "u3",
		"i1", "i2", "i3",
	}
	sensorService := fimptype.Service{
		Address: "/rt:dev/rn:oss/ad:1/sv:" + service + "/ad:" + addr,
		Name:    service,
		Groups:  []string{"ch_0"},
		Alias:   alias,
		Enabled: true,
		Props:   props,
		Interfaces: []fimptype.Interface{
			createInterface("in", "cmd.meter.get_report", "null", "1"),
			createInterface("in", "cmd.meter_ext.get_report", "null", "1"),
			createInterface("out", "evt.meter.report", "float", "1"),
			createInterface("out", "evt.meter_ext.report", "float_map", "1"),
		},
	}
	return sensorService
}

func (ns *NetworkService) MakeInclusionReport(addr string) fimptype.ThingInclusionReport {
	meterService := createMeterService(addr, "meter_elec", "meter")

	incReport := fimptype.ThingInclusionReport{
		Address:        addr,
		CommTechnology: "oss",
		ProductName:    "Oss real time meter",
		Groups:         []string{"ch_0"},
		Services: []fimptype.Service{
			meterService,
		},
		Alias:     "oss",
		ProductId: "HAN SolOss", // idk what
		DeviceId:  addr,
	}

	return incReport
}
