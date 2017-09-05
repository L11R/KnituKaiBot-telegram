package main

import (
	"bytes"
	"fmt"
	"github.com/beevik/etree"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/nfnt/resize"
	"github.com/orcaman/writerseeker"
	"github.com/unidoc/unidoc/pdf/core"
	"github.com/unidoc/unidoc/pdf/creator"
	"gopkg.in/resty.v0"
	"image"
	"strings"
)

func Link(update tgbotapi.Update) {
	url := update.Message.Text
	if strings.HasSuffix(url, "/index.html") {
		url = strings.Replace(url, "/index.html", "", 1)
	}

	resp, err := resty.R().Get(url + "/pages.xml")
	code := resp.StatusCode()

	if err != nil || code != 200 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "При запросе e-library.kai.ru что-то пошло не так. Возможно сервер недоступен.")
		bot.Send(msg)
	} else {
		msgStart := tgbotapi.NewMessage(update.Message.Chat.ID, "Начинаем загрузку страниц...")
		sent, _ := bot.Send(msgStart)

		doc := etree.NewDocument()
		doc.ReadFromBytes(resp.Body())

		root := doc.SelectElement("PageflipDataSet").SelectElement("PageOrder")

		c := creator.New()
		for i, page := range root.SelectElements("PageData") {
			attr := page.SelectAttr("LargeFile")
			resp, err = resty.R().Get(url + "/" + attr.Value)

			imgOrig, _, err := image.Decode(bytes.NewReader(resp.Body()))
			if err != nil {
				continue
			}

			imgResized := resize.Resize(612, 0, imgOrig, resize.Lanczos2)

			img, err := creator.NewImageFromGoImage(imgResized)
			if err != nil {
				continue
			}
			img.ScaleToWidth(612.0)

			height := 612.0 * img.Height() / img.Width()

			// JPEG Encoder
			encoder := core.NewDCTEncoder()
			encoder.Quality = 90
			encoder.Width = int(img.Width())
			encoder.Height = int(img.Height())

			img.SetEncoder(encoder)

			c.SetPageSize(creator.PageSize{612, height})
			c.NewPage()

			img.SetPos(0, 0)
			_ = c.Draw(img)

			fmt.Printf("Downloaded and added to document: %d\n", i+1)

			msgEdited := tgbotapi.NewEditMessageText(
				sent.Chat.ID,
				sent.MessageID,
				fmt.Sprintf("Страница №%d загружена и добавлена в PDF.", i+1),
			)
			bot.Send(msgEdited)
		}

		msgEdited := tgbotapi.NewEditMessageText(
			sent.Chat.ID,
			sent.MessageID,
			"PDF-файл успешно собран.",
		)
		bot.Send(msgEdited)

		ws := &writerseeker.WriterSeeker{}
		err = c.Write(ws)
		if err != nil {
			fmt.Println("Something go wrong!")
		}

		buf := new(bytes.Buffer)
		buf.ReadFrom(ws.BytesReader())

		file := tgbotapi.FileBytes{
			Name:  "book.pdf",
			Bytes: buf.Bytes(),
		}

		msgUpload := tgbotapi.NewMessage(update.Message.Chat.ID, "Начата загрузка файла на сервера Telegram...")
		bot.Send(msgUpload)

		msgDoc := tgbotapi.NewDocumentUpload(update.Message.Chat.ID, file)
		msgDoc.Caption = "Загрузка завершена."
		bot.Send(msgDoc)
	}
}
