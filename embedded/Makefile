MCU=atmega328p
F_CPU=16000000UL

AVR_ROOT := $(firstword $(wildcard C:/Users/tomik/AppData/Local/Arduino15/packages/arduino/tools/avr-gcc/*))
AVRD_ROOT := $(firstword $(wildcard C:/Users/tomik/AppData/Local/Arduino15/packages/arduino/tools/avrdude/*))

AVR_BIN := $(AVR_ROOT)/bin

CC = "$(AVR_BIN)/avr-gcc.exe"
OBJCOPY = "$(AVR_BIN)/avr-objcopy.exe"
AVRDUDE = "$(AVRD_ROOT)/bin/avrdude.exe"
AVRDUDECONF = "$(AVRD_ROOT)/etc/avrdude.conf"

PORT=COM3
BAUD=115200
PROGRAMMER=arduino

TARGET=main

SRC := $(wildcard *.c)
OBJ := $(SRC:.c=.o)

CFLAGS = -mmcu=$(MCU) -DF_CPU=$(F_CPU) -Os -Wall -Wextra -std=c11 -MMD -MP
LDFLAGS = -mmcu=$(MCU)

.PHONY: all flash clean

all: $(TARGET).hex

$(TARGET).elf: $(OBJ)
	$(CC) $(LDFLAGS) $^ -o $@

%.o: %.c
	$(CC) $(CFLAGS) -c $< -o $@

$(TARGET).hex: $(TARGET).elf
	$(OBJCOPY) -O ihex -R .eeprom $< $@

flash: $(TARGET).hex
	$(AVRDUDE) -C $(AVRDUDECONF) \
	-c $(PROGRAMMER) \
	-P $(PORT) \
	-b $(BAUD) \
	-p m328p \
	-D \
	-U flash:w:$(TARGET).hex:i

clean:
	del /Q *.o *.d *.elf *.hex 2>nul || ver>nul

-include $(OBJ:.o=.d)
