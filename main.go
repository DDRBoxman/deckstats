package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"sync"
	"time"

	"github.com/StackExchange/wmi"
	"github.com/karalabe/hid"
	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dimg"
)

var NUM_KEYS = 15
var ICON_SIZE = 72

var NUM_FIRST_PAGE_PIXELS = 2583
var NUM_SECOND_PAGE_PIXELS = 2601

var reset = []byte{0x0B, 0x63, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

var brightness = []byte{0x05, 0x55, 0xAA, 0xD1, 0x01, 0, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

var streamDeck *hid.Device

type OHMSensor struct {
	Name  string
	Value float32
}

func main() {
	draw2d.SetFontFolder("./fonts")

	devices := hid.Enumerate(4057, 96)

	var err error
	streamDeck, err = devices[0].Open()
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go readLoop(streamDeck)

	for {
	time.Sleep(1000)

		var dst []OHMSensor
		q := `Select * from Sensor Where (Parent LIKE "/intelcpu/[0-9]" OR Parent LIKE "/amdcpu/[0-9]") AND SensorType = "Temperature"`
		err = wmi.Query(q, &dst, ".", "root\\OpenHardwareMonitor")
		if err != nil {
			log.Fatal(err)
		}

		drawTempToKey("CPU", dst[4].Value, 4)

		q = `Select * from Sensor Where (Parent LIKE "/nvidiagpu/[0-9]" OR Parent LIKE "/atigpu/[0-9]") AND SensorType = "Temperature"`
		err = wmi.Query(q, &dst, ".", "root\\OpenHardwareMonitor")
		if err != nil {
			log.Fatal(err)
		}

		drawTempToKey("GPU", dst[0].Value, 3)
	}
	
	wg.Wait()
}

func drawTempToKey(label string, value float32, key int) {
	dest := image.NewRGBA(image.Rect(0, 0, ICON_SIZE, ICON_SIZE))
	gc := draw2dimg.NewGraphicContext(dest)

	gc.SetFillColor(color.RGBA{0xff, 0xff, 0xff, 0xff})
	gc.SetStrokeColor(color.RGBA{0xff, 0xff, 0xff, 0xff})

	gc.SetFontSize(28)
	gc.SetFontData(draw2d.FontData{
		Name: "Roboto",
	})

	gc.FillStringAt(fmt.Sprintf("%.0f°", value), 10, 32+20)

	gc.SetFontSize(8)
	gc.FillStringAt(label, 10, 72-8)

	writeImageToKey(dest, key)
}

func writeImageToKey(image *image.RGBA, key int) {
	pixels := make([]byte, ICON_SIZE*ICON_SIZE*3)

	for r := 0; r < ICON_SIZE; r++ {
		rowStartImage := r * ICON_SIZE * 4
		rowStartPixels := r * ICON_SIZE * 3
		for c := 0; c < ICON_SIZE; c++ {
			colPosImage := (c * 4) + rowStartImage
			colPosPixels := (ICON_SIZE * 3) + rowStartPixels - (c * 3) - 1

			pixels[colPosPixels-2] = image.Pix[colPosImage+2]
			pixels[colPosPixels-1] = image.Pix[colPosImage+1]
			pixels[colPosPixels] = image.Pix[colPosImage]
		}
	}

	writePage1(streamDeck, key, pixels[0:NUM_FIRST_PAGE_PIXELS*3])
	writePage2(streamDeck, key, pixels[NUM_FIRST_PAGE_PIXELS*3:])
}

func writePage1(sd *hid.Device, key int, pixels []byte) {
	header := []byte{
		0x02, 0x01, 0x01, 0x00, 0x00, (byte)(key + 1), 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x42, 0x4d, 0xf6, 0x3c, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x36, 0x00, 0x00, 0x00, 0x28, 0x00,
		0x00, 0x00, 0x48, 0x00, 0x00, 0x00, 0x48, 0x00,
		0x00, 0x00, 0x01, 0x00, 0x18, 0x00, 0x00, 0x00,
		0x00, 0x00, 0xc0, 0x3c, 0x00, 0x00, 0xc4, 0x0e,
		0x00, 0x00, 0xc4, 0x0e, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	header = append(header, pixels...)

	data := make([]byte, 8191)

	copy(data, header)

	_, err := sd.Write(data)
	if err != nil {
		log.Fatal(err)
	}
}

func writePage2(sd *hid.Device, key int, pixels []byte) {
	header := []byte{
		0x02, 0x01, 0x02, 0x00, 0x01, (byte)(key + 1),
	}

	padding := make([]byte, 10)

	header = append(header, padding...)
	header = append(header, pixels...)

	data := make([]byte, 8191)

	copy(data, header)

	_, err := sd.Write(data)
	if err != nil {
		log.Fatal(err)
	}
}

func readLoop(sd *hid.Device) {
	data := make([]byte, 255)
	for {
		time.Sleep(1000)
		size, err := sd.Read(data)
		if err != nil {
			log.Println(err)
			continue
		}

		log.Println(size)
		log.Println(data)
	}
}
