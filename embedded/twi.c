#ifndef F_CPU
#define F_CPU 16000000UL
#endif

#include "twi.h"
#include <avr/io.h>
#include <stdint.h>

#define TWI_FREQ 100000UL

void twi_init(void) {
    TWSR = 0x00;
    TWBR = (uint8_t)(((F_CPU / TWI_FREQ) - 16) / 2);
    TWCR = (1 << TWEN);
}

uint8_t twi_start(uint8_t address_rw) {
    TWCR = (1 << TWINT) | (1 << TWSTA) | (1 << TWEN);
    while (!(TWCR & (1 << TWINT))) {}

    TWDR = address_rw;
    TWCR = (1 << TWINT) | (1 << TWEN);
    while (!(TWCR & (1 << TWINT))) {}

    return 1;
}

void twi_stop(void) {
    TWCR = (1 << TWINT) | (1 << TWSTO) | (1 << TWEN);
}

uint8_t twi_write(uint8_t data) {
    TWDR = data;
    TWCR = (1 << TWINT) | (1 << TWEN);
    while (!(TWCR & (1 << TWINT))) {}
    return 1;
}
