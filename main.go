// main.go

package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

func main() {
	address := flag.String("a", "", "MAC address of the target device")
	noCheck := flag.Bool("no-check", false, "Skips image check")
	density := flag.Int("d", 2, "Printer density (1~3)")
	labelType := flag.Int("t", 1, "Label type (1~3)")
	quantity := flag.Int("n", 1, "Number of copies")
	flag.Parse()

	if *address == "" {
		fmt.Println("Error: MAC address is required.")
		return
	}

	imgFile := flag.Arg(0)
	img, err := loadImage(imgFile)
	if err != nil {
		fmt.Printf("Error loading image: %v\n", err)
		return
	}

	if img.Bounds().Max.X/img.Bounds().Max.Y > 1 {
		// Rotate clockwise 90 degrees, so the upper line (left line) prints first.
		img = rotateImage(img, 270)
	}

	*noCheck = true

	if !*noCheck && (img.Bounds().Max.X != 96 || img.Bounds().Max.Y >= 600) {
		fmt.Println("Error: Image dimensions are not valid.")
		return
	}

	printer := NewPrinterClient(*address)
	printer.SetLabelType(byte(*labelType))
	printer.SetLabelDensity(byte(*density))

	printer.StartPrint()
	printer.AllowPrintClear()
	printer.StartPagePrint()
	printer.SetDimension(img.Bounds().Max.Y, img.Bounds().Max.X)
	printer.SetQuantity(uint16(*quantity))

	packets := naiveEncoder(img)
	for _, pkt := range packets {
		printer.send(pkt)
	}

	printer.EndPagePrint()
	for printer.GetPrintStatus().Page != uint16(*quantity) {
		fmt.Println(printer.GetPrintStatus())
	}

	printer.EndPrint()
}

func loadImage(filename string) (image.Image, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func rotateImage(img image.Image, i int) image.Image {
	rotated := image.NewRGBA64(image.Rect(0, 0, img.Bounds().Dy(), img.Bounds().Dx()))
	for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			srcX := img.Bounds().Max.X - x - 1
			srcY := y
			srcColor := color.NRGBAModel.Convert(img.At(srcX, srcY)).(color.NRGBA)
			rotated.SetRGBA64(y, x, color.RGBA64(color.NRGBA64{R: uint16(srcColor.R), G: uint16(srcColor.G), B: uint16(srcColor.B), A: uint16(srcColor.A)}))
		}
	}
	return rotated
}
