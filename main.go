package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Configuration struct {
	Auth          string `json:"auth"`
	LicenseKey    string `json:"license_key"`
	PhanLoaiTrung bool   `json:"phanloaitrung"`
	AutoHatchEgg  bool   `json:"autohatchegg"`
	DoHiemTrung   int    `json:"dohiemtrung"`
	Delay         int    `json:"delaythuhoachtrung"`
}

const (
	api = "https://api.quackquack.games"
)

var (
	Auth       string
	LicenseKey string
	config     Configuration
)

type Duck struct {
	ID        int `json:"id"`
	TotalRare int `json:"total_rare"`
}

type Nest struct {
	ID int `json:"id"`
}

type NestData struct {
	Nests []Nest `json:"nest"`
	Ducks []Duck `json:"duck"`
}

type GoldenDuckResponse struct {
	Data struct {
		TimeToGoldenDuck json.Number `json:"time_to_golden_duck"`
	} `json:"data"`
}

var (
	eggHarvestedCount int
	timeToGoldenDuck  string
	goldenDuckCount   int
)

var bot *tgbotapi.BotAPI
var (
	nestIds    []int
	collecting bool
	ducks      []struct {
		ID        int `json:"id"`
		TotalRare int `json:"total_rare"`
	}
)

const telegramToken = "6739561395:AAG1iG3nS8nw2jIVoeAoGh_HDi5HpEbUG5U"
const telegramChatIDStr = "5192341527"

var telegramChatID int64

type License struct {
	Key       string `json:"key"`
	Blocked   bool   `json:"blocked"`
	Reason    string `json:"reason"`
	Timestamp string `json:"timestamp"`
}

func GetLicense(url string) (License, error) {
	var license License
	response, err := http.Get(url)
	if err != nil {
		return license, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return license, fmt.Errorf("Error getting license, status: %s", response.Status)
	}

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return license, err
	}

	err = json.Unmarshal(data, &license)
	if err != nil {
		return license, err
	}

	return license, nil
}

func CheckLicense(config Configuration) error {
	license, err := GetLicense("https://raw.githubusercontent.com/learnjavalorant/Storage-assets/main/QuackQuack.li")
	if err != nil {
		return err
	}

	timestamp, err := time.Parse("01/02/2006 15:04", license.Timestamp)
	if err != nil {
		return err
	}

	if license.Blocked {
		return fmt.Errorf("License is blocked: %s", license.Reason)
	}

	if time.Now().After(timestamp) {
		return fmt.Errorf("License expired at: %s", timestamp.Format("01/02/2006 15:04"))
	}

	if config.LicenseKey != license.Key {
		return fmt.Errorf("Invalid license key")
	}

	return nil
}

func main() {
	color.Yellow("Tool AirDrop QuackQuack...")
	color.Yellow("make by javalorant/libur... t.me/liburx")
	err := LoadConfig()
	if err != nil {
		fmt.Println("Lỗi khi load config:", err)
		return
	}
	err = CheckLicense(config)
	if err != nil {
		fmt.Println("License invalid:", err)
		return
	}
	go func() {
		for {
			time.Sleep(3 * time.Minute)
			err := CheckLicense(config)
			if err != nil {
				fmt.Println("License invalid:", err)
				return
			}
		}
	}()

	for {
		getNest()
	}
}

func LoadConfig() error {
	file, err := ioutil.ReadFile("config.json")
	if err != nil {
		return err
	}
	err = json.Unmarshal(file, &config)
	if err != nil {
		return err
	}
	Auth = config.Auth
	LicenseKey = config.LicenseKey
	return nil
}

func getNest() {
	client := &http.Client{}
	req, err := http.NewRequest("GET", api+"/nest/list-reload", nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Authorization", Auth)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error:", resp.Status)
		return
	}
	var data struct {
		Data struct {
			Nest []struct {
				ID int `json:"id"`
			} `json:"nest"`
			Duck []struct {
				ID        int `json:"id"`
				TotalRare int `json:"total_rare"`
			} `json:"duck"`
		} `json:"data"`
	}

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}
	nestIds = nil
	for _, nest := range data.Data.Nest {
		nestIds = append(nestIds, nest.ID)
	}

	ducks = data.Data.Duck

	collect()
}

func collect() {
	if collecting {
		return
	}

	collecting = true
	for _, nestID := range nestIds {
		collectDetail(nestID)
		time.Sleep(time.Duration(config.Delay) * time.Millisecond)
	}

	collecting = false
	time.Sleep(time.Duration(config.Delay) * time.Millisecond)
	go collect()
}
func collectDetail(nestID int) {
	form := url.Values{}
	form.Set("nest_id", fmt.Sprintf("%d", nestID))

	body := bytes.NewBufferString(form.Encode())

	client := &http.Client{}
	req, err := http.NewRequest("POST", api+"/nest/collect", body)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", Auth)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	var data interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Println("Error decoding collect response:", err)
		return
	}
	layEgg(nestID)
}

func layEgg(nestID int) {
	if len(ducks) == 0 {
		return
	}

	duck := ducks[0]
	ducks = ducks[1:]

	form := url.Values{}
	form.Set("nest_id", fmt.Sprintf("%d", nestID))
	form.Set("duck_id", fmt.Sprintf("%d", duck.ID))
	client := &http.Client{}
	body := bytes.NewBufferString(form.Encode())
	req, err := http.NewRequest("POST", api+"/nest/lay-egg", body)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", Auth)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error:", resp.Status)
		return
	}

	var layEggData struct {
		Data struct {
			Name      string `json:"name"`
			TotalRare int    `json:"total_rare"`
			ID        int    `json:"id"`
			Rate      int    `json:"rate"`
		} `json:"data"`
	}

	err = json.NewDecoder(resp.Body).Decode(&layEggData)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}
	eggBalance := GetBalanceEGG()
	goldenDuckTime := GetGlodenDuckTime()
	goldenDuckTimeInt, err := strconv.Atoi(goldenDuckTime)
	if err != nil {
		fmt.Println("Lỗi khi lấy thời gian xuất hiện golden duck:", err)
		return
	}
	if goldenDuckTimeInt == 0 {
		color.Green("Đã xuất hiện Gloden Duck bắt đầu đánh...")
		getInfo()
	}
	eggHarvestedCount++
	color.Yellow("\rNEST ID: %d | EGG NAME: %s | ĐỘ HIẾM: %d | ID: %d | TỶ LỆ: %d | SỐ DƯ EGG: %.2f | Golden Duck xuất hiện sau: %s\n", nestID, layEggData.Data.Name, layEggData.Data.TotalRare, layEggData.Data.ID, layEggData.Data.Rate, eggBalance, goldenDuckTime)

	if layEggData.Data.TotalRare > config.DoHiemTrung {
		if config.PhanLoaiTrung {
			//sendMessageToTelegram(fmt.Sprintf("Đã tìm thấy trứng có độ hiếm hơn 3: \nID VỊT: %d | EGG NAME: %s | ĐỘ HIẾM: %d | ID: %d | TỶ LỆ: %d | SỐ DƯ EGG: %.2f", nestID, layEggData.Data.Name, layEggData.Data.TotalRare, layEggData.Data.ID, layEggData.Data.Rate, eggBalance))
			fmt.Println("Dừng thu hoạch vì tìm ra trứng có độ hiếm trên %s...", config.DoHiemTrung)
			os.Exit(0)
			return
		} else {
			if config.AutoHatchEgg {
				go hatchEgg(nestID)
				//sendMessageToTelegram(fmt.Sprintf("Đã tìm thấy trứng có độ hiếm hơn 3: \nID VỊT: %d | EGG NAME: %s | ĐỘ HIẾM: %d | ID: %d | TỶ LỆ: %d | SỐ DƯ EGG: %.2f", nestID, layEggData.Data.Name, layEggData.Data.TotalRare, layEggData.Data.ID, layEggData.Data.Rate, eggBalance))
				fmt.Println("Bắt đầu ấp trứng...")
			}
		}
	}
}

func getInfo() {
	client := &http.Client{}
	goldenDuckTime := GetGlodenDuckTime()
	goldenDuckTimeInt, err := strconv.Atoi(goldenDuckTime)
	if err != nil {
		fmt.Println("Lỗi khi lấy thời gian xuất hiện golden duck:", err)
		return
	}
	if goldenDuckTimeInt <= 0 {
		log.Println("Đánh vịt vàng time!")

		rewardReq, err := http.NewRequest("GET", api+"/golden-duck/reward", nil)
		if err != nil {
			log.Printf("Error creating reward request: %v", err)
			return
		}
		rewardReq.Header.Set("Content-Type", "application/json")
		rewardReq.Header.Set("Authorization", Auth)

		rewardResp, err := client.Do(rewardReq)
		if err != nil {
			log.Printf("Error sending reward request: %v", err)
			return
		}
		defer rewardResp.Body.Close()

		if rewardResp.StatusCode != http.StatusOK {
			log.Printf("Error: %s", rewardResp.Status)
			return
		}

		rewardBody, err := ioutil.ReadAll(rewardResp.Body)
		if err != nil {
			log.Printf("Error reading reward response body: %v", err)
			return
		}

		log.Printf("Phần thưởng được nhận là: %s", rewardBody)
		logger("Phần thưởng được nhận là", string(rewardBody))
		form := url.Values{}
		form.Add("type", "1")
		claimReq, err := http.NewRequest("POST", api+"/golden-duck/claim", strings.NewReader(form.Encode()))
		if err != nil {
			log.Printf("Error creating claim request: %v", err)
			return
		}
		claimReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		claimReq.Header.Set("Authorization", Auth)
		claimResp, err := client.Do(claimReq)
		if err != nil {
			log.Printf("Error sending claim request: %v", err)
			return
		}
		defer claimResp.Body.Close()
		if claimResp.StatusCode == http.StatusOK {
			log.Println("Nhận thành công rồi!")
		} else {
			claimBody, err := ioutil.ReadAll(claimResp.Body)
			if err != nil {
				log.Printf("Error reading claim response body: %v", err)
				return
			}
			log.Printf("Error claiming reward: %s", claimBody)
		}
	}
}

func GetGlodenDuckTime() string {
	client := &http.Client{}
	req, err := http.NewRequest("GET", api+"/golden-duck/info", nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return "Không thể lấy được thông tin"
	}
	req.Header.Set("Authorization", Auth)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return "Không thể kết nối đến server"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error:", resp.Status)
		return "Không thể lấy được thông tin"
	}

	var data GoldenDuckResponse

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return "Không thể lấy được thông tin"
	}

	return string(data.Data.TimeToGoldenDuck)
}

func GetBalanceEGG() float64 {
	client := &http.Client{}
	req, err := http.NewRequest("GET", api+"/balance/get", nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return 0
	}
	req.Header.Set("Authorization", Auth)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error:", resp.Status)
		return 0
	}

	var data struct {
		Data struct {
			Data []struct {
				Symbol  string `json:"symbol"`
				Balance string `json:"balance"`
			} `json:"data"`
		} `json:"data"`
	}

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return 0
	}

	var eggBalance float64
	for _, item := range data.Data.Data {
		if item.Symbol == "EGG" {
			eggBalance, _ = strconv.ParseFloat(item.Balance, 64)
			break
		}
	}

	return eggBalance
}
func hatchEgg(nestID int) error {
	form := url.Values{}
	form.Set("nest_id", fmt.Sprintf("%d", nestID))

	body := bytes.NewBufferString(form.Encode())

	client := &http.Client{}
	req, err := http.NewRequest("POST", api+"/nest/hatch", body)
	if err != nil {
		return fmt.Errorf("error creating hatch request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", Auth)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending hatch request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error: %s", resp.Status)
	}

	var data struct {
		Data struct {
			TimeRemain int `json:"time_remain"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return fmt.Errorf("error decoding hatch response: %v", err)
	}

	time.Sleep(time.Duration(data.Data.TimeRemain) * time.Second)
	if err := collectDuck(nestID); err != nil {
		return fmt.Errorf("error collecting duck after hatch: %v", err)
	}

	return nil
}

func collectDuck(nestID int) error {
	form := url.Values{}
	form.Set("nest_id", fmt.Sprintf("%d", nestID))

	body := bytes.NewBufferString(form.Encode())

	client := &http.Client{}
	req, err := http.NewRequest("POST", api+"/nest/collect-duck", body)
	if err != nil {
		return fmt.Errorf("error creating collect duck request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", Auth)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending collect duck request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error: %s", resp.Status)
	}

	var data struct {
		Data interface{} `json:"data"`
	}
	fmt.Print("đã mở trứng thành công....")
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return fmt.Errorf("error decoding collect duck response: %v", err)
	}

	return nil
}

func logger(message string, result string) {
	file, err := os.OpenFile("reward.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer file.Close()

	fileLogger := log.New(file, "LOG: ", log.Ldate|log.Ltime|log.Lshortfile)
	fileLogger.Printf("%s. Result: %s\n", message, result)
	fmt.Printf("Logged message: %s. Result: %s\n", message, result)
}

func sendMessageToTelegram(message string) {
	msg := tgbotapi.NewMessage(telegramChatID, message)
	bot.Send(msg)
	//_, err := bot.Send(msg)
	//if err != nil {
	//	log.Println(err)
	//}
}
func handleStatusCommand(update tgbotapi.Update) {
	sendStatusToTelegram()
}

func setupBotCommands() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "status":
				handleStatusCommand(update)
			}
		}
	}
}
func sendStatusToTelegram() {
	goldenDuckTime := GetGlodenDuckTime()
	message := fmt.Sprintf("Số trứng đã thu hoạch: %d\nSố Golden Duck đã đánh: %d\nGolden Duck xuất hiện sau: %s", eggHarvestedCount, goldenDuckCount, goldenDuckTime)
	sendMessageToTelegram(message)
}
