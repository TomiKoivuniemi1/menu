# "Menu" project

## Project overview

Coffee place ordering system using bluetooth.

---

## Demonstration videos (Check reviewpoints.md for the order of review points covered in each)

Video 1. https://www.youtube.com/shorts/pvxUaYxWrPo

Video 2. https://www.youtube.com/shorts/IHPZXeJwYnI

Video 3. https://www.youtube.com/shorts/vsOcJsxc8LI

Video 4. https://www.youtube.com/shorts/BY6MEgGa0-s

Video 5. https://www.youtube.com/shorts/DTVPQyP0zFw

Video 6. https://www.youtube.com/shorts/cM6RdebQ2xQ

---

## Setup instructions

### Firmware flashing
- cd embedded
- make clean
- make all
- make flash

### Building the shop interface
- cd app
- go mod tidy
- go build -o app.exe

### Running the shop interface
- ./app.exe

Open browser:
- http://localhost:8080

---

## LCD user interface instructions

### Main menu

- A = Meal
- B = Lunch
- C = Drinks
- D = Cart and sending the order

---

### Meal/Lunch/Drinks

- A = previous item
- B = next item
- 1-9 = quantity
- 0 = quantity of 10
- `#` = add to cart
- `*` = back

---

### Cart

- A = previous cart item
- B = next cart item
- `#` = send order
- `*` = back

---

### Order notifications

Customer receives:
- Order sent notification
- Order accepted notification
- Order ready notification

When order is ready:
- LCD displays ready message with order number
- Passive buzzer melody is played

---

## Shop interface instructions

### Features (If you cant run the app it's okay, you can see a screenshot of it in the file *"shopinterface.png on the root)

- `Live Orders` panel  
  Displays all incoming customer orders in real time.

- `Accept` button  
  Marks the selected order as accepted and sends an acceptance notification to the customer interface.

- `Complete` button  
  Marks the selected order as completed and sends an order ready notification to the customer interface.

- `Clear All Orders` button  
  Clears the entire order list from the shop interface and customer interface.

- `Save item` button  
  Adds a new menu item or a modified one to the menu database.

- `Clear` button  
  Clears the current row

- `Edit` button  
  Move the category, name and price to "Save item" row to be edited

- `Delete` button  
  Removes the selected menu item from the menu.

- `Send Full Menu To Device` button  
  Sends the latest menu data to the customer interface with bt.

- `Serial Status` indicator  
  Displays if the Bluetooth connection is open between the shop interface and the customer interface.
---

## System architecture

### Customer interface: Arduino firmware

The Arduino firmware runs:

- LCD menu system
- keypad navigation
- menu browsing
- cart handling
- Bluetooth communication
- order status handling
- buzzer notifications

Components used:

- Arduino Uno (ATmega328P)
- 16x2 LCD with I2C backpack
- HC-06 Bluetooth module
- 4x4 keypad
- Passive buzzer

Schematic for hardware:

- `wiringdiagram.png` in the project root

### Shop interface: Go application

The Go application runs the shop staff interface backend.

It handles:

- serving the browser-based staff interface
- menu item storage using `data/menu.json`
- order storage using `data/orders.json`
- REST API endpoints for menu and order actions
- real-time browser updates using Server-Sent Events
- Bluetooth Classic communication through the HC-06 COM port
- sending menu updates to the customer interface
- receiving customer orders from the Arduino
- sending order status updates back to the Arduino

### Files architecture

- `main.go` — shop interface backend, API, order/menu storage, SSE updates, and Bluetooth serial bridge
- `main.c` — customer interface system logic and state
- `bt.c` — Bluetooth UART communication
- `buzzer.c` — ready melody generation
- `keypad.c` — keypad scanning and input handling
- `lcd.c` — LCD user interface
- `twi.c` — I2C communication
- `Makefile` — build, compile, and flash automation using avr-gcc and avrdude

---

## Technologies used

### Embedded

- C
Used to implement the entire firmware logic for UI, Bluetooth communication, cart handling and hardware interaction.

- avr-gcc (AVR GNU Compiler Collection)
Used to compile the C source code into a binary executable (.hex) for the ATmega328P microcontroller.

- avrdude (AVR Downloader/UploaDEr)
Used to upload and flash the compiled firmware onto the Arduino over USB.

- ATmega328P (Arduino Uno)
An 8-bit AVR microcontroller used as the main processor for the customer interface firmware and hardware control.

- Bluetooth Classic (HC-06)
A wireless serial communication technology used for all data exchange between the customer interface and the shop interface.

- UART (Universal Asynchronous Receiver/Transmitter)
A serial communication protocol used for communication between the HC-06 Bluetooth module and the microcontroller.

- I2C (Inter-Integrated Circuit) / TWI (Two-Wire Interface)
A two-wire communication bus used to communicate with the LCD through the I2C backpack.

- GPIO (General Purpose Input/Output)
Microcontroller pins used for reading keypad inputs and controlling outputs such as the buzzer.

### Shop interface

- Go
Used to implement the backend server logic, REST API, Bluetooth communication handling and order management.

- go.bug.st/serial
Used to handle serial Bluetooth communication over COM ports between the HC-06 module and the shop interface.

- HTTP server
Used to host the shop interface locally.

- REST API
Used to manage menu items, orders, order statuses and serial communication endpoints.

- Server-Sent Events (SSE)
Used to push real-time order updates from the Go backend to the browser interface without page refreshes.

- JavaScript
Used to implement frontend logic, API communication and real-time updates for user interface in the browser.

- JSON persistence
Used to store menu items and order history in local JSON files.

- HTML
Used to structure the shop user interface.

- CSS
Used to style the browser interface

---

## What I learned

- Bluetooth Classic communication
- Embedded LCD user interface design
- Menu and cart system implementation
- Embedded and browser application integration
- Building a functional wireless system