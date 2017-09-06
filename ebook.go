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

	// Create new document for XML/HTML
	doc := etree.NewDocument()

	// Getting Title of Book
	resp, err := resty.R().Get(url + "/index.html")
	code := resp.StatusCode()

	doc.ReadFromBytes(resp.Body())

	title := doc.SelectElement("html").SelectElement("head").SelectElement("title").Text()
	title = strings.TrimSpace(title)

	// Getting pages structure
	resp, err = resty.R().Get(url + "/pages.xml")
	code = resp.StatusCode()

	if err != nil || code != 200 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "При запросе e-library.kai.ru что-то пошло не так. Возможно сервер недоступен.")
		bot.Send(msg)
	} else {
		// Notifying user that book downloading has been started
		msgStart := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Файл: <b>%s</b>\nНачинаем загрузку страниц...", title))
		msgStart.ParseMode = "HTML"
		sent, _ := bot.Send(msgStart)

		// Read and Select new root with array of PageData
		doc.ReadFromBytes(resp.Body())
		root := doc.SelectElement("PageflipDataSet").SelectElement("PageOrder")

		c := creator.New()
		for i, page := range root.SelectElements("PageData") {
			// Select attribute with link on image file
			attr := page.SelectAttr("LargeFile")
			resp, err = resty.R().Get(url + "/" + attr.Value)

			// Load image from GET-request bytes body
			imgOrig, _, err := image.Decode(bytes.NewReader(resp.Body()))
			if err != nil {
				continue
			}

			// Resize image for file size optimization
			imgResized := resize.Resize(612, 0, imgOrig, resize.Lanczos2)

			// Load resized image as PDF component
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

			// Create new page
			c.SetPageSize(creator.PageSize{612, height})
			c.NewPage()

			// Draw image on page
			img.SetPos(0, 0)
			_ = c.Draw(img)

			// Update message to notify user that new page was downloaded and added to doc
			msgEdited := tgbotapi.NewEditMessageText(
				sent.Chat.ID,
				sent.MessageID,
				fmt.Sprintf("Файл: <b>%s</b>\nСтраница №%d загружена и добавлена в PDF.", title, i+1),
			)
			msgEdited.ParseMode = "HTML"
			bot.Send(msgEdited)
		}

		// Write PDF-file in-memory in Bytes
		ws := &writerseeker.WriterSeeker{}
		err = c.Write(ws)
		if err != nil {
			fmt.Println("Something go wrong!")
		}

		buf := new(bytes.Buffer)
		buf.ReadFrom(ws.BytesReader())

		file := tgbotapi.FileBytes{
			Name:  title + ".pdf",
			Bytes: buf.Bytes(),
		}

		// Notifying user that doc already built
		msgEdited := tgbotapi.NewEditMessageText(
			sent.Chat.ID,
			sent.MessageID,
			fmt.Sprintf("Файл: <b>%s</b>\nPDF-файл успешно собран.", title),
		)
		msgEdited.ParseMode = "HTML"
		bot.Send(msgEdited)

		// Notifying user that upload on Telegram servers has been started
		msgUpload := tgbotapi.NewMessage(update.Message.Chat.ID, "Начата загрузка файла на сервера Telegram...")
		bot.Send(msgUpload)

		// File with message about completion
		msgDoc := tgbotapi.NewDocumentUpload(update.Message.Chat.ID, file)
		msgDoc.Caption = "Загрузка завершена."
		bot.Send(msgDoc)
	}
}
