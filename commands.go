package main

import (
	"errors"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/tidwall/gjson"
	r "gopkg.in/gorethink/gorethink.v3"
	"gopkg.in/resty.v0"
	"log"
	"regexp"
	"strings"
	"time"
)

func StartCommand(update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Чтобы начать использование бота тебе достаточно сохранить свою группу командой такого вида: <code>/save 4108</code>. Разумеется можно указать любую другую группу. После этого все команды станут доступны. Команда для краткой справки по всем доступным командам: /help")
	msg.ParseMode = "HTML"
	bot.Send(msg)
}

func HelpCommand(update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "/today - расписание на сегодня.\n/tomorrow - на завтра.\n/full - на всю неделю.\n/keyboard - удобная клавиатура с днями недели.\n/remove - если вы вдруг догадались кинуть бота в чат и страдаете от того, что клавитуара появилась у всех.\n/week - показывает какая сейчас неделя (чётная или нет).\n\n/save - сохраняет вашу группу и её расписание.\n/update - обновляет расписание вашей группы.\n/delete - полностью удаляет ваш профиль из бота.\n/status - отображает текущий статус.")
	msg.ParseMode = "HTML"
	bot.Send(msg)
}

func KeyboardCommand(update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Клавиатура активирована!")
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Понедельник"),
			tgbotapi.NewKeyboardButton("Вторник"),
			tgbotapi.NewKeyboardButton("Среда"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Четверг"),
			tgbotapi.NewKeyboardButton("Пятница"),
			tgbotapi.NewKeyboardButton("Суббота"),
		),
	)
	bot.Send(msg)
}

func RemoveCommand(update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Клавиатура удалена!")
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(false)
	bot.Send(msg)
}

func WeekCommand(update tgbotapi.Update) {
	_, week := time.Now().ISOWeek()

	text := ""

	if week%2 == 0 {
		text += "<b>Нечётная неделя</b>"
	} else {
		text += "<b>Чётная неделя</b>"
	}

	text += fmt.Sprintf(" (%d)", week)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	msg.ParseMode = "HTML"
	bot.Send(msg)
}

func Update(groupNum string, userId int) error {
	// Делаем запрос, чтобы получить внутренний ID группы на основе её номера
	resp, err := resty.R().SetQueryParams(map[string]string{
		"p_p_id":          "pubStudentSchedule_WAR_publicStudentSchedule10",
		"p_p_lifecycle":   "2",
		"p_p_resource_id": "getGroupsURL",
		"query":           groupNum,
	}).Get("https://kai.ru/raspisanie")
	if err != nil {
		return err
	}

	// Достаем ID группы, из полученного JSON
	groupId := gjson.Get(resp.String(), "0.id").Num

	// Делаем запрос, чтобы получить расписание группы, на основе полученного ID
	resp, err = resty.R().SetQueryParams(map[string]string{
		"p_p_id":          "pubStudentSchedule_WAR_publicStudentSchedule10",
		"p_p_lifecycle":   "2",
		"p_p_resource_id": "schedule",
	}).SetFormData(map[string]string{
		"groupId": fmt.Sprint(groupId),
	}).Post("https://kai.ru/raspisanie")
	if err != nil {
		return err
	}

	schedule := resp.String()

	if len(schedule) > 2 {
		// Добавляем в базу пустую запись о новой группе
		_, err = r.Table("groups").Insert(map[string]interface{}{
			"id":       groupId,
			"schedule": make([]interface{}, 0),
			"time":     r.Now(),
		}, r.InsertOpts{
			Conflict: "update",
		}).RunWrite(session)
		if err != nil {
			log.Println(err)
		}

		// Цикл по дням недели
		for i := 1; i <= 6; i++ {
			dayNum := fmt.Sprint(i) + "."

			// Создаем массив для хранения занятий за день
			dayArray := make([]map[string]string, 0)

			// Цикл по занятиям
			subjectsCount := gjson.Get(schedule, dayNum+"#")
			for j := 0; j < int(subjectsCount.Int()); j++ {
				subjectNum := fmt.Sprint(j) + "."

				// Достаем все нужные поля из JSON, а затем удаляем все лишние символы
				subjectTime := strings.TrimSpace(gjson.Get(schedule, dayNum+subjectNum+"dayTime").Str)
				subjectWeek := strings.TrimSpace(gjson.Get(schedule, dayNum+subjectNum+"dayDate").Str)
				subjectName := strings.TrimSpace(gjson.Get(schedule, dayNum+subjectNum+"disciplName").Str)
				subjectType := strings.TrimSpace(gjson.Get(schedule, dayNum+subjectNum+"disciplType").Str)
				buildNum := strings.TrimSpace(gjson.Get(schedule, dayNum+subjectNum+"buildNum").Str)
				cabinetNum := strings.TrimSpace(gjson.Get(schedule, dayNum+subjectNum+"audNum").Str)
				teacherName := strings.TrimSpace(gjson.Get(schedule, dayNum+subjectNum+"prepodName").Str)

				// Добавляем к существующему массиву новое занятие
				dayArray = append(dayArray, map[string]string{
					"subjectTime": subjectTime,
					"subjectWeek": subjectWeek,
					"subjectName": subjectName,
					"subjectType": subjectType,
					"buildNum":    buildNum,
					"cabinetNum":  cabinetNum,
					"teacherName": teacherName,
				})
			}

			// Добавляем в базу день
			_, err := r.Table("groups").Get(groupId).Update(map[string]interface{}{
				"schedule": r.Row.Field("schedule").InsertAt(i-1, dayArray),
			}).RunWrite(session)
			if err != nil {
				log.Println(err)
			}
		}

		// Добавляем в базу запись о пользователе
		_, err = r.Table("users").Insert(map[string]interface{}{
			"id":       userId,
			"groupId":  groupId,
			"groupNum": groupNum,
		}, r.InsertOpts{
			Conflict: "update",
		}).RunWrite(session)
		if err != nil {
			log.Println(err)
		}

		return nil
	} else {
		return errors.New("Too short schedule!")
	}
}

func SaveCommand(update tgbotapi.Update) {
	re := regexp.MustCompile(`\s(.+)`)

	groupNum := re.FindStringSubmatch(update.Message.Text)
	if len(groupNum) > 0 {
		err := Update(groupNum[1], update.Message.From.ID)
		if err != nil {
			log.Println(err)
		}

		if err == nil {
			var msg tgbotapi.MessageConfig
			if err == nil {
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Cохранено!")
			} else {
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "В процессе сохранения группы что-то пошло не так... Возможно сервер с актуальным расписанием недоступен.")
			}
			bot.Send(msg)
		} else {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Похоже введен неверный номер группы.")
			bot.Send(msg)
		}
	} else {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Пример: <code>/save 4108</code>, чтобы сохранить группу 4108. Замените этот номер на любой другой.")
		msg.ParseMode = "HTML"
		bot.Send(msg)
	}
}

func GetDayName(num int) string {
	switch num {
	case 0:
		return "Понедельник"
	case 1:
		return "Вторник"
	case 2:
		return "Среда"
	case 3:
		return "Четверг"
	case 4:
		return "Пятница"
	case 5:
		return "Суббота"
	case 6:
		return "Воскресенье"
	default:
		return ""
	}
}

func GetDayNum(name string) int {
	switch name {
	case "Понедельник":
		return 0
	case "Вторник":
		return 1
	case "Среда":
		return 2
	case "Четверг":
		return 3
	case "Пятница":
		return 4
	case "Суббота":
		return 5
	case "Воскресенье":
		return 6
	default:
		return -1
	}
}

func GetDayText(subjects []map[string]string) string {
	_, week := time.Now().ISOWeek()
	text := ""

	// Цикл по занятиям
	for _, elem := range subjects {
		// Пропускаем четные пары в нечетные недели
		if week%2 == 0 && elem["subjectWeek"] == "чет" {
			continue
		}

		// Пропускаем нечетные пары в четные недели
		if week%2 != 0 && elem["subjectWeek"] == "неч" {
			continue
		}

		// Добавляем к существующему сообщению новое занятие
		if elem["subjectTime"] != "" {
			text += fmt.Sprintf("<i>%s", elem["subjectTime"])
		} else {
			text += "<i>TIME UNDEFINED"
		}

		if elem["subjectWeek"] != "" {
			text += fmt.Sprintf(" %s</i>\n", elem["subjectWeek"])
		} else {
			text += "</i>\n"
		}

		if elem["subjectName"] != "" {
			text += fmt.Sprintf("%s\n", elem["subjectName"])
		} else {
			text += "SUBJECT NAME UNDEFINED\n"
		}

		if elem["subjectType"] != "" {
			text += fmt.Sprintf("%s", elem["subjectType"])
		}

		if elem["buildNum"] != "" {
			text += fmt.Sprintf(", %s", elem["buildNum"])
		}

		if elem["cabinetNum"] != "" {
			text += fmt.Sprintf(", %s", elem["cabinetNum"])
		}

		if elem["teacherName"] != "" {
			text += fmt.Sprintf(", %s", elem["teacherName"])
		}

		text += "\n\n"
	}

	return text
}

func FullCommand(update tgbotapi.Update) {
	// Получаем из базы нужную информацию
	user, err := GetUser(update.Message.From.ID)
	if err != nil {
		log.Println(err)
	}

	group, err := GetGroup(user.GroupID)
	if err != nil {
		log.Println(err)
	}

	if err == nil {
		// Инициализируем пустое сообщение
		text := ""

		// Цикл по дням недели
		for i := range group.Schedule {
			// Добавляем к существующему сообщению день недели
			text += "<b>" + GetDayName(i) + "</b>\n"
			text += GetDayText(group.Schedule[i])
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
		msg.ParseMode = "HTML"
		bot.Send(msg)
	} else {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Что-то пошло не так... Похоже ты ещё не сохранил свою группу.")
		bot.Send(msg)
	}
}

func TodayCommand(update tgbotapi.Update) {
	// Получаем номер текущего дня
	day := int(time.Now().Weekday()) - 1

	if day != 6 {
		// Получаем из базы нужную информацию
		user, err := GetUser(update.Message.From.ID)
		if err != nil {
			log.Println(err)
		}

		group, err := GetGroup(user.GroupID)
		if err != nil {
			log.Println(err)
		}

		if err == nil {
			// Инициализируем пустое сообщение
			text := ""

			text += "<b>" + GetDayName(day) + "</b>\n"
			text += GetDayText(group.Schedule[day])

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
			msg.ParseMode = "HTML"
			bot.Send(msg)
		} else {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Что-то пошло не так... Похоже ты ещё не сохранил свою группу.")
			bot.Send(msg)
		}
	} else {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Похоже сегодня воскресенье.")
		bot.Send(msg)
	}
}

func TomorrowCommand(update tgbotapi.Update) {
	// Получаем номер завтрашнего дня
	day := int(time.Now().Weekday())

	if day != 6 {
		// Получаем из базы нужную информацию
		user, err := GetUser(update.Message.From.ID)
		if err != nil {
			log.Println(err)
		}

		group, err := GetGroup(user.GroupID)
		if err != nil {
			log.Println(err)
		}

		if err == nil {
			// Инициализируем пустое сообщение
			text := ""

			text += "<b>" + GetDayName(day) + "</b>\n"
			text += GetDayText(group.Schedule[day])

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
			msg.ParseMode = "HTML"
			bot.Send(msg)
		} else {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Что-то пошло не так... Похоже ты ещё не сохранил свою группу.")
			bot.Send(msg)
		}
	} else {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Похоже завтра воскресенье.")
		bot.Send(msg)
	}
}

func GetCommand(update tgbotapi.Update) {
	day := GetDayNum(update.Message.Text)

	if day < 6 {
		// Получаем из базы нужную информацию
		user, err := GetUser(update.Message.From.ID)
		if err != nil {
			log.Println(err)
		}

		group, err := GetGroup(user.GroupID)
		if err != nil {
			log.Println(err)
		}

		if err == nil {
			// Инициализируем пустое сообщение
			text := ""

			text += "<b>" + GetDayName(int(day)) + "</b>\n"
			text += GetDayText(group.Schedule[day])

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
			msg.ParseMode = "HTML"
			bot.Send(msg)
		} else {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Что-то пошло не так... Похоже ты ещё не сохранил свою группу.")
			bot.Send(msg)
		}
	} else {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Любопытный? Да, мне было не лень обработать и этот кейс.")
		bot.Send(msg)
	}
}

func StatusCommand(update tgbotapi.Update) {
	// Получаем из базы нужную информацию
	user, err := GetUser(update.Message.From.ID)
	if err != nil {
		log.Println(err)
	}

	group, err := GetGroup(user.GroupID)
	if err != nil {
		log.Println(err)
	}

	if err == nil {
		// Инициализируем пустое сообщение
		text := ""

		text += "<b>ID:</b> " + fmt.Sprint(user.Id) + "\n"
		text += "<b>Группа:</b> " + user.GroupNum + "\n"
		text += "<b>Последнее обновление:</b> " + fmt.Sprint(group.Time) + "\n"

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
		msg.ParseMode = "HTML"
		bot.Send(msg)
	} else {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Что-то пошло не так... Похоже ты ещё не сохранил свою группу.")
		bot.Send(msg)
	}
}

func UpdateCommand(update tgbotapi.Update) {
	// Получаем из базы нужную информацию
	user, err := GetUser(update.Message.From.ID)
	if err != nil {
		log.Println(err)
	}

	if err == nil {
		err = Update(fmt.Sprint(user.GroupNum), user.Id)
		if err != nil {
			log.Println(err)
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Обновлено!")
		msg.ParseMode = "HTML"
		bot.Send(msg)
	} else {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "В процессе обновления расписания твоей группы что-то пошло не так... Возможно сервер с актуальным расписанием недоступен.")
		bot.Send(msg)
	}
}

func DeleteCommand(update tgbotapi.Update) {
	_, err := r.Table("users").Get(update.Message.Chat.ID).Delete().RunWrite(session)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "В процессе удаления твоего профиля из базы что-то пошло не так... Попробуй позже.")
		bot.Send(msg)
	} else {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Удалено!")
		msg.ParseMode = "HTML"
		bot.Send(msg)
	}
}
