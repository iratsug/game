package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
	"math/rand"
)

type Player struct {
	Name      string
	Health    int
	MaxHealth int
	Damage    int
	Money     int
	Wins      int
	Potion    int
}

type GameMessage struct {
	Type    string 
	Sender  string
	Content string
	Data    interface{}
}

var chatMessages = make(chan string, 10)
var chatClients = make(map[net.Conn]bool)
var chatMode = false

func main() {
	rand.Seed(time.Now().UnixNano())
	reader := bufio.NewReader(os.Stdin)
	
	for {
	
		fmt.Println("\n" + strings.Repeat("=", 40))
		fmt.Println("     АРЕНА: БИТВА ЗА СВОБОДУ")
		fmt.Println(strings.Repeat("=", 40))
		fmt.Println("1. Одиночная игра (против ботов)")
		fmt.Println("2.  Сетевая PvP битва (игрок против игрока)")
		fmt.Println("3.  Чат (общение с игроками)")
		fmt.Println("4.  Выйти")
		fmt.Println()
		fmt.Print("Выбери режим (1-4): ")
		
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		
		switch choice {
		case "1":
			singlePlayer(reader)
		case "2":
			networkPvP(reader)
		case "3":
			startChat(reader)
		case "4":
			fmt.Println("Пока! Заходи ещё!")
			return
		default:
			fmt.Println("Неверный выбор. Попробуй снова.")
		}
	}
}


func startChat(reader *bufio.Reader) {
	fmt.Println("\n" + strings.Repeat("=", 40))
	fmt.Println(" ЧАТ-КОМНАТА")
	fmt.Println(strings.Repeat("=", 40))
	fmt.Println("1.  Создать чат-комнату (сервер)")
	fmt.Println("2.  Подключиться к чату (клиент)")
	fmt.Println("3.  Назад")
	fmt.Println()
	fmt.Print("Выбери (1-3): ")
	
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	
	switch choice {
	case "1":
		startChatServer(reader)
	case "2":
		connectToChat(reader)
	default:
		return
	}
}

func startChatServer(reader *bufio.Reader) {
	fmt.Print("Введите порт для чата (например 8080): ")
	port, _ := reader.ReadString('\n')
	port = strings.TrimSpace(port)
	
	if port == "" {
		port = "8080"
	}
	
	fmt.Print("Введите своё имя в чате: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	
	if name == "" {
		name = "Админ"
	}
	
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Ошибка запуска чата:", err)
		return
	}
	defer listener.Close()
	
	fmt.Printf("\n Чат запущен на порту %s\n", port)
	fmt.Println("Ожидание подключений...")
	fmt.Println("(для выхода введи '/exit')")
	fmt.Println()
	
	go broadcastChatMessages()
	
	go serverChatInput(name)
	
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Ошибка подключения:", err)
			continue
		}
		
		chatClients[conn] = true
		
		go handleChatClient(conn)

		chatMessages <- fmt.Sprintf(" Новый участник подключился!")
	}
}

func connectToChat(reader *bufio.Reader) {
	fmt.Print("Введите адрес сервера (например 127.0.0.1:8080): ")
	address, _ := reader.ReadString('\n')
	address = strings.TrimSpace(address)
	
	if address == "" {
		address = "127.0.0.1:8080"
	}
	
	fmt.Print("Введите своё имя в чате: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	
	if name == "" {
		name = "Гость"
	}
	
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Println("Ошибка подключения к чату:", err)
		return
	}
	defer conn.Close()
	
	fmt.Printf("\n Подключился к чату %s\n", address)
	fmt.Println("(для выхода введи '/exit')")
	fmt.Println()
	
	conn.Write([]byte(name + "\n"))
	
	messages := make(chan string, 5)
	
	go func() {
		buffer := make([]byte, 1024)
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				messages <- " Отключился от сервера"
				close(messages)
				return
			}
			messages <- string(buffer[:n])
		}
	}()
	
	go func() {
		for msg := range messages {
			fmt.Println(msg)
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		if text == "/exit" {
			break
		}
		conn.Write([]byte(name + ": " + text + "\n"))
	}
}

func handleChatClient(conn net.Conn) {
	defer func() {
		delete(chatClients, conn)
		conn.Close()
		chatMessages <- " Участник покинул чат"
	}()
	
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return
	}
	name := strings.TrimSpace(string(buffer[:n]))
	
	chatMessages <- fmt.Sprintf(" %s присоединился к чату", name)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			break
		}
		
		msg := string(buffer[:n])
		chatMessages <- msg
	}
}

func broadcastChatMessages() {
	for msg := range chatMessages {
		fmt.Println(msg) 

		for conn := range chatClients {
			_, err := conn.Write([]byte(msg + "\n"))
			if err != nil {
				delete(chatClients, conn)
				conn.Close()
			}
		}
	}
}

func serverChatInput(name string) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		if text == "/exit" {
			os.Exit(0)
		}
		chatMessages <- name + ": " + text
	}
}

func networkPvP(reader *bufio.Reader) {
	fmt.Println("\n" + strings.Repeat("=", 40))
	fmt.Println(" СЕТЕВОЙ PVP-РЕЖИМ")
	fmt.Println(strings.Repeat("=", 40))
	fmt.Println("1.  Создать игру (сервер)")
	fmt.Println("2.  Подключиться к игре (клиент)")
	fmt.Println("3.  Назад")
	fmt.Println()
	fmt.Print("Выбери (1-3): ")
	
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	
	switch choice {
	case "1":
		startGameServer(reader)
	case "2":
		connectToGame(reader)
	default:
		return
	}
}

func startGameServer(reader *bufio.Reader) {
	fmt.Print("Введите порт для игры (например 9090): ")
	port, _ := reader.ReadString('\n')
	port = strings.TrimSpace(port)
	
	if port == "" {
		port = "9090"
	}
	
	fmt.Print("Введите имя бойца: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	player1 := Player{
		Name:      name,
		Health:    120,
		MaxHealth: 120,
		Damage:    20,
		Money:     0,
		Wins:      0,
		Potion:    2,
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Ошибка создания игры:", err)
		return
	}
	defer listener.Close()
	
	fmt.Printf("\n Игра создана на порту %s\n", port)
	fmt.Println("Ожидание противника...")

	conn, err := listener.Accept()
	if err != nil {
		fmt.Println("Ошибка подключения противника:", err)
		return
	}
	defer conn.Close()
	
	fmt.Println(" Противник подключился!")
	fmt.Println("Битва начинается через 3 секунды...")
	time.Sleep(3 * time.Second)

	decoder := gob.NewDecoder(conn)
	var player2 Player
	err = decoder.Decode(&player2)
	if err != nil {
		fmt.Println("Ошибка получения данных противника")
		return
	}
	
	fmt.Printf("\n ПРОТИВНИК: %s\n", player2.Name)
	fmt.Printf(" Здоровье: %d |  Урон: %d |  Зелий: %d\n", 
		player2.Health, player2.Damage, player2.Potion)
	fmt.Println()

	pvpBattle(conn, &player1, &player2, reader, true)
}

func connectToGame(reader *bufio.Reader) {
	fmt.Print("Введите адрес сервера (например 127.0.0.1:9090): ")
	address, _ := reader.ReadString('\n')
	address = strings.TrimSpace(address)
	
	fmt.Print("Введите имя бойца: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	player2 := Player{
		Name:      name,
		Health:    120,
		MaxHealth: 120,
		Damage:    20,
		Money:     0,
		Wins:      0,
		Potion:    2,
	}

	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Println("Ошибка подключения к игре:", err)
		return
	}
	defer conn.Close()
	
	fmt.Println(" Подключился к серверу!")

	encoder := gob.NewEncoder(conn)
	err = encoder.Encode(player2)
	if err != nil {
		fmt.Println("Ошибка отправки данных")
		return
	}
	
	fmt.Println("Ожидание начала битвы...")

	decoder := gob.NewDecoder(conn)
	var player1 Player
	err = decoder.Decode(&player1)
	if err != nil {
		fmt.Println("Ошибка получения данных противника")
		return
	}
	
	fmt.Printf("\n ПРОТИВНИК: %s\n", player1.Name)
	fmt.Printf(" Здоровье: %d |  Урон: %d |  Зелий: %d\n", 
		player1.Health, player1.Damage, player1.Potion)
	fmt.Println()

	pvpBattle(conn, &player1, &player2, reader, false)
}

func pvpBattle(conn net.Conn, player1, player2 *Player, reader *bufio.Reader, isServer bool) {
	encoder := gob.NewEncoder(conn)
	decoder := gob.NewDecoder(conn)
	
	var myPlayer, enemyPlayer *Player
	
	if isServer {
		myPlayer = player1
		enemyPlayer = player2
	} else {
		myPlayer = player2
		enemyPlayer = player1
	}
	
	round := 1
	
	for myPlayer.Health > 0 && enemyPlayer.Health > 0 {
		fmt.Printf("\n" + strings.Repeat("=", 30))
		fmt.Printf("\n РАУНД %d \n", round)
		fmt.Println(strings.Repeat("=", 30))
		fmt.Printf("ТВОЁ ЗДОРОВЬЕ:  %d/%d\n", myPlayer.Health, myPlayer.MaxHealth)
		fmt.Printf("ЗДОРОВЬЕ %s:  %d\n", enemyPlayer.Name, enemyPlayer.Health)
		fmt.Printf("ТВОИ ЗЕЛЬЯ:  %d\n", myPlayer.Potion)
		fmt.Println()

		fmt.Println("1.  Атаковать")
		fmt.Println("2.  Выпить зелье (+30 HP)")
		fmt.Println("3.  Предложить ничью")
		fmt.Print("Твой ход: ")
		
		action, _ := reader.ReadString('\n')
		action = strings.TrimSpace(action)
		
		var actionType string
		var actionData interface{}
		
		switch action {
		case "1":
	
			damage := rand.Intn(myPlayer.Damage) + 5
			actionType = "attack"
			actionData = damage
			
			enemyPlayer.Health -= damage
			fmt.Printf(" Ты нанёс %d урона!\n", damage)
			
		case "2":
			if myPlayer.Potion > 0 {
				heal := 30
				myPlayer.Health = min(myPlayer.Health+heal, myPlayer.MaxHealth)
				myPlayer.Potion--
				actionType = "heal"
				actionData = heal
				fmt.Printf(" Ты выпил зелье! +%d HP\n", heal)
			} else {
				fmt.Println(" Нет зелий! Ход пропущен")
				actionType = "skip"
			}
			
		case "3":
			fmt.Print("Предложить ничью? (да/нет): ")
			confirm, _ := reader.ReadString('\n')
			confirm = strings.TrimSpace(confirm)
			
			if confirm == "да" {
				actionType = "draw"
			} else {
				continue
			}
			
		default:
			fmt.Println("Неверный выбор, ход пропущен")
			actionType = "skip"
		}

		msg := GameMessage{
			Type:    actionType,
			Sender:  myPlayer.Name,
			Content: fmt.Sprintf("%s сделал ход", myPlayer.Name),
			Data:    actionData,
		}
		encoder.Encode(msg)

		if actionType == "draw" {
			fmt.Println("\n Ты предложил ничью. Ожидание ответа...")

			var response GameMessage
			err := decoder.Decode(&response)
			if err != nil {
				fmt.Println("Противник отключился")
				return
			}
			
			if response.Type == "accept_draw" {
				fmt.Println("\n Противник согласился на ничью!")
				fmt.Println("Битва закончена вничью!")
				return
			} else {
				fmt.Println(" Противник отказался от ничьей!")
				continue
			}
		}

		if enemyPlayer.Health <= 0 {
			fmt.Printf("\n ТЫ ПОБЕДИЛ %s! \n", enemyPlayer.Name)
			

			result := GameMessage{
				Type:   "victory",
				Sender: myPlayer.Name,
			}
			encoder.Encode(result)
			return
		}

		fmt.Println("\n Ожидание хода противника...")
		
		var enemyMsg GameMessage
		err := decoder.Decode(&enemyMsg)
		if err != nil {
			fmt.Println(" Противник отключился!")
			fmt.Println("ТЫ ПОБЕДИЛ ТЕХНИЧЕСКОЙ ПОБЕДОЙ!")
			return
		}

		switch enemyMsg.Type {
		case "attack":
			damage := enemyMsg.Data.(int)
			myPlayer.Health -= damage
			fmt.Printf(" %s атаковал тебя на %d урона!\n", enemyPlayer.Name, damage)
			
		case "heal":
			heal := enemyMsg.Data.(int)
			enemyPlayer.Health += heal
			if enemyPlayer.Health > enemyPlayer.MaxHealth {
				enemyPlayer.Health = enemyPlayer.MaxHealth
			}
			fmt.Printf(" %s выпил зелье!\n", enemyPlayer.Name)
			
		case "draw":
			fmt.Printf("\n %s предлагает ничью. Согласиться? (да/нет): ", enemyPlayer.Name)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(answer)
			
			response := GameMessage{Type: "reject_draw"}
			if answer == "да" {
				response.Type = "accept_draw"
				encoder.Encode(response)
				fmt.Println("\n Ничья принята! Битва окончена!")
				return
			} else {
				encoder.Encode(response)
				fmt.Println(" Ты отказался от ничьей!")
			}
		}

		if myPlayer.Health <= 0 {
			fmt.Printf("\n ТЫ ПРОИГРАЛ %s! \n", enemyPlayer.Name)

			result := GameMessage{
				Type:   "defeat",
				Sender: myPlayer.Name,
			}
			encoder.Encode(result)
			return
		}
		
		round++
		time.Sleep(1 * time.Second)
	}
}

func singlePlayer(reader *bufio.Reader) {
	fmt.Print("Введи имя бойца: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	
	fmt.Printf("\n%s, ты попал в плен к жестокому клану 'Черные псы'.\n", name)
	fmt.Println("Чтобы выжить, ты должен сражаться на арене.")
	fmt.Println("Побеждай врагов, зарабатывай деньги и становись сильнее!")
	fmt.Println("После 5 побед ты получишь свободу!")
	fmt.Println()

	player := Player{
		Name:      name,
		Health:    100,
		MaxHealth: 100,
		Damage:    15,
		Money:     0,
		Wins:      0,
		Potion:    1,
	}

	for {
		if player.Health <= 0 {
			fmt.Println("\n ТЫ ПРОИГРАЛ... ")
			fmt.Printf(" Побед: %d\n", player.Wins)
			return
		}
		
		if player.Wins >= 5 {
			fmt.Println("\n ТЫ ПОБЕДИЛ! ПОЛУЧИЛ СВОБОДУ! ")
			return
		}
	
		fmt.Println("\n" + strings.Repeat("=", 30))
		fmt.Printf(" КАЗАРМА (Бой %d из 5)\n", player.Wins+1)
		fmt.Println(strings.Repeat("=", 30))
		fmt.Printf(" HP: %d/%d\n", player.Health, player.MaxHealth)
		fmt.Printf(" Урон: %d\n", player.Damage)
		fmt.Printf(" Деньги: %d\n", player.Money)
		fmt.Printf(" Зелья: %d\n", player.Potion)
		fmt.Printf(" Побед: %d/5\n", player.Wins)
		
		fmt.Println("\n1.  НА АРЕНУ")
		fmt.Println("2.  Лавка")
		fmt.Println("3.  Отдохнуть")
		fmt.Println("4.  В главное меню")
		fmt.Print("Выбери: ")
		
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		
		switch choice {
		case "1":
			fightBot(&player, reader)
		case "2":
			shop(&player, reader)
		case "3":
			rest(&player)
		case "4":
			return
		}
	}
}

func fightBot(player *Player, reader *bufio.Reader) {
	var enemy Enemy
	
	switch player.Wins {
	case 0:
		enemy = Enemy{"Новичок", 50, 8, 50}
	case 1:
		enemy = Enemy{"Головорез", 70, 12, 80}
	case 2:
		enemy = Enemy{"Наёмник", 90, 15, 120}
	case 3:
		enemy = Enemy{"Гладиатор", 110, 18, 180}
	case 4:
		enemy = Enemy{"ЧЕМПИОН", 150, 25, 300}
	}

	fmt.Printf("\n БОЙ С %s \n", enemy.Name)
	fmt.Printf("Противник:  %d HP,  %d урона\n", enemy.Health, enemy.Damage)
	
	for player.Health > 0 && enemy.Health > 0 {
		fmt.Printf("\nТы:  %d | %s:  %d\n", player.Health, enemy.Name, enemy.Health)
		fmt.Println("1.  Атаковать")
		fmt.Println("2.  Зелье (+30 HP)")
		fmt.Print("Твой ход: ")
		
		action, _ := reader.ReadString('\n')
		action = strings.TrimSpace(action)
		
		switch action {
		case "1":
	
			playerDamage := rand.Intn(player.Damage) + 5
			enemy.Health -= playerDamage
			fmt.Printf(" Ты нанёс %d урона!\n", playerDamage)
			
			if enemy.Health <= 0 {
				fmt.Printf("\n %s ПОВЕРЖЕН!\n", enemy.Name)
				player.Wins++
				player.Money += enemy.Reward
				fmt.Printf(" +%d монет\n", enemy.Reward)
				
				if rand.Intn(100) < 30 {
					player.Potion++
					fmt.Println(" Нашёл зелье!")
				}
				return
			}

			enemyDamage := rand.Intn(enemy.Damage) + 3
			player.Health -= enemyDamage
			fmt.Printf(" %s бьёт на %d урона!\n", enemy.Name, enemyDamage)
			
		case "2":
			if player.Potion > 0 {
				player.Health = min(player.Health+30, player.MaxHealth)
				player.Potion--
				fmt.Println(" +30 HP")
			} else {
				fmt.Println(" Нет зелий!")
			}
		}
	}
}

func shop(player *Player, reader *bufio.Reader) {
	fmt.Println("\n ЛАВКА")
	fmt.Printf(" Деньги: %d\n", player.Money)
	fmt.Println("1.  +5 урона - 150 монет")
	fmt.Println("2.  +20 макс. HP - 100 монет")
	fmt.Println("3.  Зелье - 50 монет")
	fmt.Println("4.  Лечение (+30) - 40 монет")
	fmt.Println("5. Выйти")
	fmt.Print("Выбери: ")
	
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	
	switch choice {
	case "1":
		if player.Money >= 150 {
			player.Money -= 150
			player.Damage += 5
			fmt.Println(" Урон увеличен!")
		} else {
			fmt.Println(" Не хватает!")
		}
	case "2":
		if player.Money >= 100 {
			player.Money -= 100
			player.MaxHealth += 20
			player.Health += 20
			fmt.Println(" HP увеличено!")
		} else {
			fmt.Println(" Не хватает!")
		}
	case "3":
		if player.Money >= 50 {
			player.Money -= 50
			player.Potion++
			fmt.Println(" Куплено зелье!")
		} else {
			fmt.Println(" Не хватает!")
		}
	case "4":
		if player.Money >= 40 && player.Health < player.MaxHealth {
			player.Money -= 40
			player.Health = min(player.Health+30, player.MaxHealth)
			fmt.Println(" Подлечился!")
		} else {
			fmt.Println(" Нельзя!")
		}
	}
}

func rest(player *Player) {
	fmt.Println("\n Отдых...")
	time.Sleep(2 * time.Second)
	
	if player.Health < player.MaxHealth {
		player.Health = min(player.Health+20, player.MaxHealth)
		fmt.Println(" +20 HP")
	}
	
	if rand.Intn(100) < 30 {
		found := rand.Intn(30) + 10
		player.Money += found
		fmt.Printf(" Нашёл %d монет!\n", found)
	}
}

type Enemy struct {
	Name   string
	Health int
	Damage int
	Reward int
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
