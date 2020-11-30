package main

import (
	"flag"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"time"

	"fmt"

	"uk.ac.bris.cs/gameoflife/stubs"
)

var FinalWorld [][]uint8
var CurrentWorld [][]uint8
var AliveCells int
var Currentturn int
var done bool

type NextStateOperation struct{}

// Distributor divides the work between workers and interacts with other goroutines.
func distributor(world [][]uint8, turns, threads int) {
	done = false
	World := world
	height := len(World)
	width := len(World[0])
	rem := mod(height, threads)
	splitThreads := height / threads

	AliveCells = 0
	for h := 0; h < height; h++ {
		for g := 0; g < width; g++ {
			if World[h][g] == alive {
				AliveCells++
			}
		}
	}

	for Currentturn = 0; Currentturn < turns; Currentturn++ {
		workerChannels := make([]chan [][]uint8, threads)
		for i := range workerChannels {
			workerChannels[i] = make(chan [][]uint8)
			startY := i*splitThreads + rem
			endY := (i+1)*splitThreads + rem

			if i < rem {
				startY = i * (splitThreads + 1)
				endY = (i + 1) * (splitThreads + 1)
			}
			go worker(height, width, startY, endY, World, workerChannels[i])
		}

		tempWorld := make([][]uint8, 0)
		for i := range workerChannels { // collects the resulting parts into a single 2D slice
			workerResults := <-workerChannels[i]
			tempWorld = append(tempWorld, workerResults...)
		}
		World = tempWorld
		CurrentWorld = World
		AliveCells = 0
		for h := 0; h < height; h++ {
			for g := 0; g < width; g++ {
				if World[h][g] == alive {
					AliveCells++
				}
			}
		}
	}
	FinalWorld = World
	done = true
}

func worker(height, width, startY, endY int, world [][]byte, out chan<- [][]uint8) {
	newData := calculateNextState(height, width, startY, endY, world)
	out <- newData
}

//Initial state of the world
func (s *NextStateOperation) InitialState(req stubs.Request, res *stubs.Response) (err error) {
	fmt.Println("Gamestate initialised")
	World := req.Message
	Turn := req.Turns
	Threads := req.Threads
	go distributor(World, Turn, Threads)
	return
}

//Final state of the world
func (s *NextStateOperation) FinalState(req stubs.Request, res *stubs.Response) (err error) {
	fmt.Println("Final Gamestate returned")
	for done == false {
		//
	}
	res.Message = FinalWorld
	return
}

//Return current World + Turn for counting alive cells
func (s *NextStateOperation) Alive(req stubs.Request, res *stubs.Response) (err error) {
	fmt.Println("Return num of alive cells")
	res.Turn = Currentturn
	res.AliveCells = AliveCells
	return
}

func (s *NextStateOperation) DoKeypresses(req stubs.Request, res *stubs.Response) (err error) {
	fmt.Println("Return num of alive cells")
	res.Turn = Currentturn
	res.Message = CurrentWorld
	return
}

////////////
const alive = 255
const dead = 0

func mod(x, m int) int {
	return (x + m) % m
}

//calculates number of neighbours of cell
func calculateNeighbours(height, width, x, y int, world [][]byte) int {
	neighbours := 0
	for i := -1; i <= 1; i++ {
		for j := -1; j <= 1; j++ {
			if i != 0 || j != 0 { //not [y][x]
				if world[mod(y+i, height)][mod(x+j, width)] == alive {
					neighbours++
				}
			}
		}
	}
	return neighbours
}

//takes the current state of the world and completes one evolution of the world. It then returns the result.
func calculateNextState(height, width, startY, endY int, world [][]byte) [][]byte {
	//makes a new world
	newWorld := make([][]byte, endY-startY)
	for i := range newWorld {
		newWorld[i] = make([]byte, width)
	}
	//sets cells to dead or alive according to num of neighbours
	for y := startY; y < endY; y++ {
		for x := 0; x < width; x++ {
			neighbours := calculateNeighbours(height, width, x, y, world)
			if world[y][x] == alive {
				if neighbours == 2 || neighbours == 3 {
					newWorld[y-startY][x] = alive
				} else {
					newWorld[y-startY][x] = dead
				}
			} else {
				if neighbours == 3 {
					newWorld[y-startY][x] = alive
				} else {
					newWorld[y-startY][x] = dead
				}
			}
		}
	}
	return newWorld
}

func main() {
	pAddr := flag.String("port", ":8030", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	rpc.Register(&NextStateOperation{})
	listener, err := net.Listen("tcp", *pAddr)
	if err != nil {
		log.Fatal("listen error:", err)
	}

	defer listener.Close()
	rpc.Accept(listener)
}
