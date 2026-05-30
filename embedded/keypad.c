#ifndef F_CPU
#define F_CPU 16000000UL
#endif

#include "keypad.h"
#include <avr/io.h>
#include <util/delay.h>
#include <stdint.h>

static const char keymap[4][4] = {
    {'1','2','3','A'},
    {'4','5','6','B'},
    {'7','8','9','C'},
    {'*','0','#','D'}
};

void keypad_init(void) {
    DDRB |= (1 << PB5) | (1 << PB4) | (1 << PB3) | (1 << PB2);
    PORTB |= (1 << PB5) | (1 << PB4) | (1 << PB3) | (1 << PB2);

    DDRB &= ~((1 << PB1) | (1 << PB0));
    PORTB |= (1 << PB1) | (1 << PB0);

    DDRD &= ~((1 << PD7) | (1 << PD6));
    PORTD |= (1 << PD7) | (1 << PD6);
}

static void set_row_low(uint8_t row) {
    PORTB |= (1 << PB5) | (1 << PB4) | (1 << PB3) | (1 << PB2);
    switch (row) {
        case 0: PORTB &= ~(1 << PB5); break;
        case 1: PORTB &= ~(1 << PB4); break;
        case 2: PORTB &= ~(1 << PB3); break;
        case 3: PORTB &= ~(1 << PB2); break;
    }
}

static uint8_t read_col(uint8_t col) {
    switch (col) {
        case 0: return (PINB & (1 << PB1)) == 0;
        case 1: return (PINB & (1 << PB0)) == 0;
        case 2: return (PIND & (1 << PD7)) == 0;
        case 3: return (PIND & (1 << PD6)) == 0;
    }
    return 0;
}

char keypad_getkey(void) {
    for (uint8_t r = 0; r < 4; r++) {
        set_row_low(r);
        _delay_us(5);
        for (uint8_t c = 0; c < 4; c++) {
            if (read_col(c)) {
                _delay_ms(20);
                if (read_col(c)) {
                    while (read_col(c)) {}
                    PORTB |= (1 << PB5) | (1 << PB4) | (1 << PB3) | (1 << PB2);
                    return keymap[r][c];
                }
            }
        }
    }
    PORTB |= (1 << PB5) | (1 << PB4) | (1 << PB3) | (1 << PB2);
    return 0;
}
