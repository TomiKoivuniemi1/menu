#ifndef F_CPU
#define F_CPU 16000000UL
#endif

#include "buzzer.h"
#include <avr/io.h>
#include <util/delay.h>
#include <stdint.h>

#define BUZZER_DDR  DDRC
#define BUZZER_PORT PORTC
#define BUZZER_PIN  PC3

static void tone_delay_us(uint16_t half_period_us, uint16_t duration_ms) {
    uint32_t cycles = ((uint32_t)duration_ms * 1000UL) / ((uint32_t)half_period_us * 2UL);
    for (uint32_t i = 0; i < cycles; i++) {
        BUZZER_PORT |= (1 << BUZZER_PIN);
        for (uint16_t j = 0; j < half_period_us; j++) _delay_us(1);
        BUZZER_PORT &= ~(1 << BUZZER_PIN);
        for (uint16_t j = 0; j < half_period_us; j++) _delay_us(1);
    }
}

void buzzer_init(void) {
    BUZZER_DDR |= (1 << BUZZER_PIN);
    BUZZER_PORT &= ~(1 << BUZZER_PIN);
}

void buzzer_beep_ok(void) {
    tone_delay_us(1000, 80);
}

void buzzer_ready_tune(void) {
    tone_delay_us(956, 120);
    _delay_ms(40);
    tone_delay_us(758, 120);
    _delay_ms(40);
    tone_delay_us(638, 180);
}
