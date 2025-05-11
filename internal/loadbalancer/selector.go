package loadbalancer

import "net/url"

// Интерфейс сервера
type Backend interface {
	URL() *url.URL // Адрес
	Alive() bool   // Состояние жив/мертв
	SetAlive(bool) // Установка состояния
	Inc()          // +1 к количеству активных соединений
	Done()         // -1 при завершении запроса
	Conns() int64  // Получить текущее количество соединений
}

// Интерфейс выбора сервера
type Selector interface {
	Next() Backend // возвращает выбранный сервер
}
