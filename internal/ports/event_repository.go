package ports

import (
	"context"

	"github.com/rachelJG/event-notification-service/internal/core/domain"
)

type EventRepository interface {
	Create(ctx context.Context, event domain.Event) (string, error)
}

//los port en arquitectura hexagonal van en la capa core/ports
// es normalmente una interfaz y es usada para entrar y salir del sistema.
// hay dos tipos de port: input y output
// input: es la interfaz que recibe la aplicacion (use case)
// output: es la interfaz que envia la aplicacion (repository)
//core = logica de negocio, si el negocio fuera un restaurante, el core fuera la cocina
// ports = son las reglas para hablar con la cocina (core) son interfaces (contratos) hay de entrada y salida
// los adapters = meseros  proveedores , puede ser grpc, postgres etc
