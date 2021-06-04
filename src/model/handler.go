package model

import (
	"github.com/futurehomeno/fimpgo"
)

type OssHandler struct {
	mqtt *fimpgo.MqttTransport
}

func (oh *OssHandler) MakeMeterReportMsg(addr string, value float64, unit string, oldMsg *fimpgo.FimpMessage) (*fimpgo.FimpMessage, *fimpgo.Address) {
	service := "meter_elec"
	props := make(map[string]string)
	props["unit"] = unit
	msg := fimpgo.NewMessage("evt.meter.report", "meter_elec", "float", value, props, nil, oldMsg)
	adr, _ := fimpgo.NewAddressFromString("pt:j1/mt:evt/rt:dev/rn:oss/ad:1/sv:" + service + "/ad:" + addr)
	return msg, adr
}

func (oh *OssHandler) MakeMeterExtendedReportMsg(addr string, value map[string]float64, oldMsg *fimpgo.FimpMessage) (*fimpgo.FimpMessage, *fimpgo.Address) {
	service := "meter_elec"
	msg := fimpgo.NewFloatMapMessage("evt.meter_ext.report", "meter_elec", value, nil, nil, oldMsg)
	adr, _ := fimpgo.NewAddressFromString("pt:j1/mt:evt/rt:dev/rn:oss/ad:1/sv:" + service + "/ad:" + addr)
	return msg, adr
}
