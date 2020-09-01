//v0.1
package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	tgbotapi "gopkg.in/telegram-bot-api.v4"

	//"reflect"
	"bytes"
	"os/exec"
	"strconv"
	"strings"
)

type data struct {
	Tasks []tasks
	Notes []notes
}
type tasks struct {
	Id        bson.ObjectId `_id`
	Task      string
	Completed string
}
type notes struct {
	Id     bson.ObjectId
	TaskId bson.ObjectId `_id`
	Note   string
}
type message struct {
	Task      string
	Messageid int
}

func toCorrectHTML(w string) string {
	r := strings.NewReplacer("<br>", "\n", "&nbsp;", " ", "</div><div>", "\n", "<div>", "", "</div>", "", "<span>", "", "</span>", "")
	return r.Replace(w)
}
func toCorrectLink(w string) string {
	r := strings.NewReplacer("<a href", "</b><a href", "</a>", "</a><b>")
	return r.Replace(w)
}

func main() {
	//Привязываем бота
	bot, err := tgbotapi.NewBotAPI("627449834:AAEa7SvtKbo7e7RVOf6GA4c33QoKMlhyqcg") //delowebbot
	if err != nil {
		log.Panic(err)
	}
	//chatId := int64(-1001120618082)
	//chatId := int64(-235896196) //Ситуационный центр Губернатора
	chatId := int64(476580)

	//Привязываем mongodb к сайту
	session, err := mgo.Dial("localhost")
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()
	session.SetMode(mgo.Monotonic, true) //http://stackoverflow.com/questions/38572332/compare-consistency-models-used-in-mgo
	c := session.DB("local").C("totalmanager")

	dir := "web"
	tasks := []tasks{}
	notes := []notes{}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err = c.Find(nil).Sort("_id").All(&tasks)
		if err != nil {
			log.Fatal(err)
		}
		//db.getCollection('totalmanager').aggregate([{$unwind:"$notes"}, {$project:{"id":"$notes.id", "note":"$notes.note"}}])
		err = c.Pipe([]bson.M{{"$unwind": "$notes"}, {"$project": bson.M{"id": "$notes.id", "note": "$notes.note"}}}).All(&notes)
		if err != nil {
			log.Fatal(err)
		}

		data := data{Tasks: tasks, Notes: notes}
		t, _ := template.ParseFiles(dir + "/index.html")
		t.Execute(w, data)
	})

	http.HandleFunc("/addFile", func(w http.ResponseWriter, r *http.Request) {
		//fmt.Println(r.FormValue("name"))
		//fmt.Println(r.FormValue("lastModified"))
		lm := r.FormValue("lastModified")
		i, _ := strconv.ParseInt(lm[:len(lm)-3], 10, 64)
		//timeFrom := time.Unix(i-1, 0).Format("2006-01-02T15:04:05Z")  //Во времени UTC
		//timeTo := time.Unix(i+1, 0).Format("2006-01-02T15:04:05Z")
		timeFrom := time.Unix(i-1, 0).Format("2006-01-02 15:04:05") //Во локальном времени
		timeTo := time.Unix(i+1, 0).Format("2006-01-02 15:04:05")
		find, err := exec.Command("/bin/sh", "-c", "find $HOME -name '"+r.FormValue("name")+"'"+" -newermt '"+timeFrom+"' ! -newermt '"+timeTo+"'").CombinedOutput()
		if err != nil {
			log.Println(err)
		}
		//fmt.Println("find -name '" + r.FormValue("name") + "'" + " -newermt '" + timeFrom + "' ! -newermt '" + timeTo + "'")
		//fmt.Println(string(find))
		fmt.Fprintf(w, strings.Split(string(find), "\n")[0]) //Посылаем клиенту путь до первого найденного файла
	})

	//Не работает при запуске из systemctl start totalmanager.service
	//Возможно, стоило просто прописать Environment=PATH=/bin:/usr/bin:/sbin:/usr/sbin в файле totalmanager.service
	http.HandleFunc("/openFile", func(w http.ResponseWriter, r *http.Request) {
		cmd := exec.Command("/bin/sh", "-c", "xdg-open '"+r.FormValue("link")+"'")
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		}
	})

	http.HandleFunc("/updateTaskStatus", func(w http.ResponseWriter, r *http.Request) {
		err = c.Update(bson.M{"_id": bson.ObjectIdHex(r.FormValue("Id"))}, bson.M{"$set": bson.M{"completed": r.FormValue("Completed")}})
		if err != nil {
			log.Fatal(err)
		}
	})
	http.HandleFunc("/updateTask", func(w http.ResponseWriter, r *http.Request) {
		//fmt.Println("Тип переменной:", reflect.TypeOf(r.FormValue("Id")).Kind())
		err = c.Update(bson.M{"_id": bson.ObjectIdHex(r.FormValue("Id"))}, bson.M{"$set": bson.M{"task": r.FormValue("Task")}})
		if err != nil {
			log.Fatal(err)
		}
	})
	http.HandleFunc("/updateNote", func(w http.ResponseWriter, r *http.Request) {
		err = c.Update(bson.M{"notes.id": bson.ObjectIdHex(r.FormValue("Id"))}, bson.M{"$set": bson.M{"notes.$.note": r.FormValue("Note")}})
		if err != nil {
			log.Fatal(err)
		}
		if r.FormValue("Tg") != "true" {
			return
		} //Если кнопка Telegram не нажата, прерываем функцию..
		//Ищем ChatID и MessageID в базе данных
		//db.getCollection('totalmanager').aggregate([{$unwind: "$notes"}, { $match: {"notes.id": ObjectId("5b9f783f2903af05f2faa0f7")}}, {$unwind: "$notes.chats"}, { $match: {"notes.chats.chatId":476580}}, {$project : {task: "$task", messageid:"$notes.chats.messageId"} }])
		//Project не может корректно обрабатывать большие буквы в середине текста - нельзя указывать messageId
		message := message{}
		err = c.Pipe([]bson.M{{"$unwind": "$notes"}, {"$match": bson.M{"notes.id": bson.ObjectIdHex(r.FormValue("Id"))}}, {"$unwind": "$notes.chats"}, {"$match": bson.M{"notes.chats.chatId": chatId}}, {"$project": bson.M{"task": "$task", "messageid": "$notes.chats.messageId"}}}).One(&message)
		//Формуруем сообщение для Telegram
		if err != nil { //Если не находим, отправляем новое сообщение и добавляем ChatID и MessageID в базу данных
			c.Find(bson.M{"notes.id": bson.ObjectIdHex(r.FormValue("Id"))}).Select(bson.M{"task": 1}).One(&message) //Чтобы отловить Task
			//В Telegram API есть ограничение - нельзя сделать так <b>Текст<a href=...>Ссылка</a></b>
			//Поэтому можно либо очистить теги ссылки https://play.golang.org/p/lwkU5jGla_Z, либо добавить закрывающий и открывающий теги </b> и <b>
			text := toCorrectHTML("<b>" + toCorrectLink(message.Task) + "</b>\n\n" + r.FormValue("Note"))
			newMessage := tgbotapi.NewMessage(chatId, text)
			newMessage.ParseMode = "HTML"
			m, err := bot.Send(newMessage)
			if err == nil { //Если ошибок нет, то m.Chat.ID и m.MessageID будут определены, можем обновлять запись.
				c.Update(bson.M{"notes.id": bson.ObjectIdHex(r.FormValue("Id"))}, bson.M{"$push": bson.M{"notes.$.chats": bson.M{"chatId": m.Chat.ID, "messageId": m.MessageID}}})
			}
		} else { //Если находим, редактируем сообщение бота
			text := toCorrectHTML("<b>" + toCorrectLink(message.Task) + "</b>\n\n" + r.FormValue("Note"))
			newEditMessageText := tgbotapi.NewEditMessageText(chatId, message.Messageid, text)
			newEditMessageText.ParseMode = "HTML"
			bot.Send(newEditMessageText)
		}
	})

	http.HandleFunc("/addTask", func(w http.ResponseWriter, r *http.Request) {
		objId := bson.NewObjectId()
		err = c.Insert(bson.M{"_id": objId, "createdTime": time.Now(), "completed": "no"})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(w, objId.Hex()) //Пересылаем клиенту уникальный objectId
	})
	http.HandleFunc("/addNote", func(w http.ResponseWriter, r *http.Request) {
		objId := bson.NewObjectId()
		err = c.Update(bson.M{"_id": bson.ObjectIdHex(r.FormValue("Id"))}, bson.M{"$push": bson.M{"notes": bson.M{"id": objId}}})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(w, objId.Hex()) //Пересылаем клиенту уникальный objectId
	})
	http.HandleFunc("/removeTask", func(w http.ResponseWriter, r *http.Request) {
		err = c.Remove(bson.M{"_id": bson.ObjectIdHex(r.FormValue("Id"))})
		if err != nil {
			log.Fatal(err)
		}
	})
	http.HandleFunc("/removeNote", func(w http.ResponseWriter, r *http.Request) {
		err = c.Update(bson.M{"_id": bson.ObjectIdHex(r.FormValue("TaskId"))}, bson.M{"$pull": bson.M{"notes": bson.M{"id": bson.ObjectIdHex(r.FormValue("Id"))}}})
		if err != nil {
			log.Fatal(err)
		}
	})
	http.Handle("/js/", http.FileServer(http.Dir(dir)))
	http.Handle("/css/", http.FileServer(http.Dir(dir)))
	http.Handle("/fonts/", http.FileServer(http.Dir(dir)))

	fmt.Println("Сервер успешно запущен!")
	http.ListenAndServe(":8000", nil)
	//http.ListenAndServeTLS(":8443", dir+"/https_cert.pem", dir+"/https_pkey.pem", nil)  //если systemd запускается от user (см. ps aux), 443 порт не поднимется
}
