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

	"github.com/DDRBoxman/deckstats/floatbuffer"
)

var NUM_KEYS = 15
var ICON_SIZE = 72

var NUM_FIRST_PAGE_PIXELS = 2583
var NUM_SECOND_PAGE_PIXELS = 2601

var reset = []byte{0x0B, 0x63, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

var brightness = []byte{0x05, 0x55, 0xAA, 0xD1, 0x01, 0, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

var streamDeck *hid.Device

var cpuTemps *floatbuffer.Buffer
var gpuTemps *floatbuffer.Buffer

type OHMSensor struct {
	Name  string
	Value float32
}

func main() {
	draw2d.SetFontFolder("./fonts")

	var err error
	cpuTemps, err = floatbuffer.NewBuffer(288) // store 4 seconds per pixel (almost 5 minutes)
	if err != nil {
		log.Fatal(err)
	}

	gpuTemps, err = floatbuffer.NewBuffer(288) // store 4 seconds per pixel (almost 5 minutes)
	if err != nil {
		log.Fatal(err)
	}

	devices := hid.Enumerate(4057, 96)

	streamDeck, err = devices[0].Open()
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go readLoop(streamDeck)

	for {
		time.Sleep(2000)

		var dst []OHMSensor
		q := `Select * from Sensor Where (Parent LIKE "/intelcpu/[0-9]" OR Parent LIKE "/amdcpu/[0-9]") AND SensorType = "Temperature"`
		err = wmi.Query(q, &dst, ".", "root\\OpenHardwareMonitor")
		if err != nil {
			log.Fatal(err)
		}

		drawTempToKey("CPU", dst[4].Value, 4)

		cpuTemps.Write(dst[4].Value)

		drawTempGraphToKey(cpuTemps, 9)

		q = `Select * from Sensor Where (Parent LIKE "/nvidiagpu/[0-9]" OR Parent LIKE "/atigpu/[0-9]") AND SensorType = "Temperature"`
		err = wmi.Query(q, &dst, ".", "root\\OpenHardwareMonitor")
		if err != nil {
			log.Fatal(err)
		}

		drawTempToKey("GPU", dst[0].Value, 3)

		gpuTemps.Write(dst[0].Value)

		drawTempGraphToKey(gpuTemps, 8)
	}
	
	wg.Wait()
}

func drawTempGraphToKey(tempBuffer *floatbuffer.Buffer, key int) {
	dest := image.NewRGBA(image.Rect(0, 0, ICON_SIZE, ICON_SIZE))
	gc := draw2dimg.NewGraphicContext(dest)

	gc.SetStrokeColor(color.RGBA{0xff, 0x00, 0x00, 0xff})
	gc.SetLineWidth(1)

	temps := tempBuffer.Floats()

	for i := 0; i<len(temps); i+=4 {
		reading := temps[i:i+4]
		
		var total float32 = 0
		for _, value:= range reading {
			total += value
		}

		gc.MoveTo(float64(i / 4), float64(ICON_SIZE))
		gc.LineTo(float64(i / 4),  float64(ICON_SIZE) - float64(total / float32(4)))
		gc.Stroke()
	}

	writeImageToKey(dest, key)
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

	gc.FillStringAt(fmt.Sprintf("%.0f°", value), 10, 32+12)

	gc.SetFontSize(14)
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
