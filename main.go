/*******************************************************************************
 * Audio Morse Decoder 'CW4ISR' (forked from Project ZyLO since 2023 July 15th)
 * Released under the MIT License (or GPL v3 until 2021 Oct 28th) (see LICENSE)
 * Univ. Tokyo Amateur Radio Club Development Task Force (https://nextzlog.dev)
*******************************************************************************/

package main

import (
	"encoding/binary"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dop251/goja"
	"github.com/gen2brain/malgo"
	"github.com/nextzlog/cw4i/core"
	"github.com/nextzlog/cw4i/util"
	"math"
	"os"
)

const (
	SQL_MIN = 1
	SQL_MAX = 100
)

const (
	DEV = "dev"
	SQL = "sql"
)

func main() {
	app := app.NewWithID("cw4i")
	cfg := malgo.ContextConfig{}
	app.Settings().SetTheme(theme.DarkTheme())
	ctx, _ := malgo.InitContext(nil, cfg, nil)
	win := app.NewWindow("CW4ISR Morse Decoder")
	history := new(History)
	capture := Capture{
		Context: ctx.Context,
		Handler: history.Add,
	}
	sql := widget.NewSlider(SQL_MIN, SQL_MAX)
	sql.OnChanged = capture.SetSquelch
	sql.SetValue(app.Preferences().Float(SQL))
	sel := capture.CanvasObject()
	his := history.CanvasObject()
	dev := app.Preferences().String(DEV)
	if dev == "" {
		sel.SetSelectedIndex(0)
	} else {
		sel.SetSelected(dev)
	}
	out := container.NewBorder(sel, sql, nil, nil, his)
	win.Resize(fyne.NewSize(640, 480))
	win.SetContent(out)
	win.ShowAndRun()
	ctx.Uninit()
	app.Preferences().SetFloat(SQL, sql.Value)
	app.Preferences().SetString(DEV, sel.Selected)
	return
}

func Script(rate int) (decoder core.Decoder, err error) {
	decoder = core.DefaultDecoder(rate)
	vm := goja.New()
	vm.Set("call", util.Call)
	vm.Set("plot", util.Plot)
	vm.Set("decoder", decoder)
	code, _ := os.ReadFile("cw4i.js")
	if _, err = vm.RunString(string(code)); err == nil {
		err = vm.ExportTo(vm.Get("decoder"), &decoder)
	}
	return
}

type Capture struct {
	Context malgo.Context
	Capture *malgo.Device
	Decoder core.Decoder
	Squelch float64
	Handler func([]core.Message)
}

func (c *Capture) SetSquelch(level float64) {
	c.Decoder.Squelch = math.Pow(10, level)
	c.Squelch = level
}

func (c *Capture) Run(dev malgo.DeviceInfo) (err error) {
	cfg := malgo.DefaultDeviceConfig(malgo.Capture)
	cfg.PeriodSizeInMilliseconds = 200
	cfg.Capture.Format = malgo.FormatS32
	cfg.Capture.DeviceID = dev.ID.Pointer()
	cfg.Capture.Channels = 1
	endian := binary.LittleEndian
	dcb := malgo.DeviceCallbacks{
		Data: func(out, in []byte, size uint32) {
			signal := make([]float64, size)
			for n := 0; n < len(in); n += 4 {
				v := endian.Uint32(in[n : n+4])
				signal[n/4] = float64(int32(v))
			}
			c.Handler(c.Decoder.Read(signal))
		},
	}
	c.Capture, _ = malgo.InitDevice(c.Context, cfg, dcb)
	c.Decoder, err = Script(int(c.Capture.SampleRate()))
	c.SetSquelch(c.Squelch)
	c.Capture.Start()
	return
}

func (c *Capture) CanvasObject() (ui *widget.Select) {
	devices, _ := c.Context.Devices(malgo.Capture)
	sel := widget.NewSelect(nil, func(name string) {
		if c.Capture != nil {
			c.Capture.Uninit()
			c.Capture = nil
		}
		for _, dev := range devices {
			if dev.Name() == name {
				c.Run(dev)
			}
		}
	})
	for _, dev := range devices {
		sel.Options = append(sel.Options, dev.Name())
	}
	return sel
}

type History struct {
	core.History
}

func (h *History) length() int {
	return len(h.Items)
}

func (h *History) canvas() fyne.CanvasObject {
	return widget.NewLabel("")
}

func (h *History) update(id int, obj fyne.CanvasObject) {
	item := h.Items[len(h.Items)-id-1]
	label := obj.(*widget.Label)
	label.SetText(item.Text)
}

func (h *History) CanvasObject() (ui fyne.CanvasObject) {
	list := widget.NewList(h.length, h.canvas, h.update)
	h.Added = func() {
		list.Refresh()
	}
	return list
}
