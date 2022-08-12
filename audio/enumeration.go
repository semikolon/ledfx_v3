package audio

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"ledfx/config"
	"ledfx/logger"
	"strings"
	"text/tabwriter"

	"github.com/gordonklaus/portaudio"
)

/*
Creates a hash of hostapi idx and device name
This ID should be the same regardless of device idx, meaning
it won't change when other audio devices are added or removed
*/
func createId(i int, n string) string {
	s := fmt.Sprintf("%d %s", i, n)
	id := sha1.New()
	id.Write([]byte(s))
	return hex.EncodeToString(id.Sum(nil))
}

func GetPaDeviceInfo(ad config.AudioDevice) (d *portaudio.DeviceInfo, err error) {
	hs, err := portaudio.HostApis()
	if err != nil {
		return
	}
	for i, h := range hs {
		for _, d := range h.Devices {
			// if d.MaxInputChannels < 1 {
			// 	continue
			// }
			if ad.Id == createId(i, d.Name) {
				return d, nil
			}
		}
	}
	logger.Logger.Warn("Saved audio input device cannot be found. Reverting to default device.")
	d, err = portaudio.DefaultInputDevice()
	if err != nil {
		return &portaudio.DeviceInfo{}, err
	}
	return d, err
}

func GetAudioDevices() (infos []config.AudioDevice, err error) {
	err = portaudio.Initialize()
	if err != nil {
		logger.Logger.Error(err)
		return
	}
	defer portaudio.Terminate()

	hs, err := portaudio.HostApis()
	if err != nil {
		logger.Logger.Error(err)
		return
	}
	for i, h := range hs {
		for _, d := range h.Devices {
			// if d.MaxInputChannels < 1 {
			// 	continue
			// }
			ad := config.AudioDevice{
				Id:          createId(i, d.Name),
				HostApi:     h.Name,
				SampleRate:  d.DefaultSampleRate,
				Name:        d.Name,
				ChannelsIn:  d.MaxInputChannels,
				ChannelsOut: d.MaxOutputChannels,
				IsDefault:   d.Name == h.DefaultInputDevice.Name || d.Name == h.DefaultOutputDevice.Name,
			}
			if ad.ChannelsIn > 2 {
				ad.ChannelsIn = 2
			}
			if ad.ChannelsOut > 2 {
				ad.ChannelsOut = 2
			}
			infos = append(infos, ad)
		}
	}
	return infos, err
}

func LogAudioDevices() {
	infos, err := GetAudioDevices()
	if err != nil {
		logger.Logger.Error(err)
		return
	}
	w := tabwriter.NewWriter(logger.Logger.Out, 1, 1, 1, ' ', 0)

	var icon rune

	for _, info := range infos {
		if info.IsDefault {
			icon = '✓'
		} else {
			icon = '⨯'
		}
		fmt.Fprintf(w, "%s:\t%s,\tChan In: %d,\tChan Out: %d,\tsamplerate: %f,\tdefault: %c\n",
			info.HostApi, info.Name, info.ChannelsIn, info.ChannelsOut, info.SampleRate, icon)
	}
	w.Flush()
}

func GetDeviceByID(id string) (config.AudioDevice, error) {
	devices, err := GetAudioDevices()
	if err != nil {
		return config.AudioDevice{}, err
	}
	for _, device := range devices {
		if device.Id == id {
			return device, nil
		}
	}

	idList := make([]string, len(devices))
	for i := range devices {
		idList[i] = devices[i].Id
	}

	// In case the ID is actually the device name
	tryByName, err := GetDeviceByName(id)
	if err != nil {
		logger.Logger.WithField("context", "Device Lookup").Warnf("Device ID/Name lookup failed!")
	} else {
		logger.Logger.WithField("context", "Device Lookup").Infof("Device ID lookup failed, but a name lookup worked!")
		return tryByName, nil
	}

	return config.AudioDevice{}, fmt.Errorf("could not find audio device matching id '%s' out of [%s]", id, strings.Join(idList, ", "))
}

func GetDeviceByName(name string) (config.AudioDevice, error) {
	devices, err := GetAudioDevices()
	if err != nil {
		return config.AudioDevice{}, err
	}
	for _, device := range devices {
		if strings.EqualFold(device.Name, name) {
			return device, nil
		}
	}

	nameList := make([]string, len(devices))
	for i := range devices {
		nameList[i] = devices[i].Name
	}

	return config.AudioDevice{}, fmt.Errorf("could not find audio device matching id '%s' out of [%s]", name, strings.Join(nameList, ", "))
}
