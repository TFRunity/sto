package main

import (
	"net/url"

	"github.com/AlexeBabanov/sto/chassis/config"
)

type Config struct {
	config.Base

	JWTSecret    config.Secret `env:"JWT_SECRET,notEmpty"`
	UsersURL     url.URL       `env:"USERS_URL,notEmpty"`
	OrdersURL    url.URL       `env:"ORDERS_URL,notEmpty"`
	InventoryURL url.URL       `env:"INVENTORY_URL,notEmpty"`
	VehicleURL   url.URL       `env:"VEHICLE_URL"`
}
