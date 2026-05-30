#ifndef F_CPU
#define F_CPU 16000000UL
#endif

#include "bt.h"
#include <avr/io.h>
#include <stdint.h>

#define BT_BAUD 9600UL
#define UBRR_VALUE ((F_CPU / 16UL / BT_BAUD) - 1UL)

void bt_init(void) {
    UBRR0H = (uint8_t)(UBRR_VALUE >> 8);
    UBRR0L = (uint8_t)(UBRR_VALUE);

    UCSR0A = 0;
    UCSR0B = (1 << RXEN0) | (1 << TXEN0);
    UCSR0C = (1 << UCSZ01) | (1 << UCSZ00);
}

void bt_putc(char c) {
    while (!(UCSR0A & (1 << UDRE0))) {}
    UDR0 = (uint8_t)c;
}

void bt_puts(const char *s) {
    if (!s) return;
    while (*s) {
        bt_putc(*s++);
    }
}

uint8_t bt_available(void) {
    return (UCSR0A & (1 << RXC0)) != 0;
}

uint8_t bt_read_byte(uint8_t *out) {
    if (!out) return 0;
    if (!bt_available()) return 0;

    *out = UDR0;
    return 1;
}