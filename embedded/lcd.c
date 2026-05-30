#ifndef F_CPU
#define F_CPU 16000000UL
#endif

#include "lcd.h"
#include "twi.h"
#include <util/delay.h>
#include <stdint.h>

#define LCD_ADDR 0x3F
#define LCD_BACKLIGHT 0x08
#define LCD_EN 0x04
#define LCD_RW 0x02
#define LCD_RS 0x01

static void lcd_expander_write(uint8_t data) {
    twi_start((LCD_ADDR << 1) | 0);
    twi_write(data | LCD_BACKLIGHT);
    twi_stop();
}

static void lcd_pulse_enable(uint8_t data) {
    lcd_expander_write(data | LCD_EN);
    _delay_us(1);
    lcd_expander_write(data & ~LCD_EN);
    _delay_us(50);
}

static void lcd_write4(uint8_t nibble, uint8_t mode) {
    uint8_t data = (nibble & 0xF0) | mode;
    lcd_expander_write(data);
    lcd_pulse_enable(data);
}

static void lcd_send(uint8_t value, uint8_t mode) {
    lcd_write4(value & 0xF0, mode);
    lcd_write4((value << 4) & 0xF0, mode);
}

static void lcd_command(uint8_t cmd) {
    lcd_send(cmd, 0);
}

void lcd_putc(char c) {
    lcd_send((uint8_t)c, LCD_RS);
}

void lcd_init(void) {
    twi_init();
    _delay_ms(50);

    lcd_write4(0x30, 0);
    _delay_ms(5);
    lcd_write4(0x30, 0);
    _delay_us(150);
    lcd_write4(0x30, 0);
    lcd_write4(0x20, 0);

    lcd_command(0x28);
    lcd_command(0x08);
    lcd_clear();
    lcd_command(0x06);
    lcd_command(0x0C);
}

void lcd_clear(void) {
    lcd_command(0x01);
    _delay_ms(2);
}

void lcd_goto(uint8_t row, uint8_t col) {
    uint8_t addr = (row == 0) ? 0x00 : 0x40;
    addr += col;
    lcd_command(0x80 | addr);
}

void lcd_puts(const char *s) {
    if (!s) return;
    while (*s) {
        lcd_putc(*s++);
    }
}

void lcd_print_fixed_16(const char *text) {
    uint8_t count = 0;

    while (text && *text && count < 16) {
        lcd_putc(*text++);
        count++;
    }

    while (count < 16) {
        lcd_putc(' ');
        count++;
    }
}

void lcd_puts_padded(const char *s) {
    lcd_print_fixed_16(s);
}

static void lcd_print_uint16(uint16_t value) {
    char buf[6];
    uint8_t i = 0;

    if (value == 0) {
        lcd_putc('0');
        return;
    }

    while (value > 0 && i < sizeof(buf)) {
        buf[i++] = (char)('0' + (value % 10));
        value /= 10;
    }

    while (i > 0) {
        lcd_putc(buf[--i]);
    }
}

void lcd_print_price(uint16_t price_cents) {
    uint16_t euros = price_cents / 100;
    uint16_t cents = price_cents % 100;

    lcd_print_uint16(euros);
    lcd_putc('.');
    lcd_putc((char)('0' + (cents / 10)));
    lcd_putc((char)('0' + (cents % 10)));
    lcd_putc('e');
}

void lcd_show_category_screen(void) {
    lcd_clear();
    lcd_goto(0, 0);
    lcd_print_fixed_16("A=Meal B=Lunch");
    lcd_goto(1, 0);
    lcd_print_fixed_16("C=Drink D=Dess");
}

void lcd_show_home_screen(void) {
    lcd_show_category_screen();
}

void lcd_show_item_screen(const char *name, uint16_t price_cents, uint8_t qty) {
    lcd_clear();

    lcd_goto(0, 0);
    lcd_print_fixed_16(name);

    lcd_goto(1, 0);
    lcd_print_price(price_cents);
    lcd_putc(' ');
    lcd_putc('x');
    if (qty > 9) {
        lcd_print_uint16(qty);
    } else {
        lcd_putc((char)('0' + qty));
    }
    lcd_putc(' ');
    lcd_putc('#');
    lcd_putc('=');
    lcd_putc('O');
    lcd_putc('K');
}

void lcd_show_quantity_screen(const char *name, uint8_t qty) {
    lcd_clear();

    lcd_goto(0, 0);
    lcd_print_fixed_16(name);

    lcd_goto(1, 0);
    lcd_puts("Qty:");
    if (qty > 99) {
        lcd_print_uint16(qty);
    } else if (qty > 9) {
        lcd_print_uint16(qty);
    } else {
        lcd_putc((char)('0' + qty));
    }
    lcd_puts(" #=add *=back");
}

void lcd_show_cart_screen(uint8_t item_count, uint16_t total_cents) {
    lcd_clear();

    lcd_goto(0, 0);
    lcd_puts("Cart items:");
    lcd_print_uint16(item_count);

    lcd_goto(1, 0);
    lcd_puts("Total ");
    lcd_print_price(total_cents);
}

void lcd_show_checkout_screen(uint16_t total_cents) {
    lcd_clear();

    lcd_goto(0, 0);
    lcd_puts("Checkout total");

    lcd_goto(1, 0);
    lcd_print_price(total_cents);
    lcd_puts(" #=send");
}

void lcd_show_order_sent_screen(void) {
    lcd_clear();
    lcd_goto(0, 0);
    lcd_print_fixed_16("Order sent");
    lcd_goto(1, 0);
    lcd_print_fixed_16("Wait confirm...");
}

void lcd_show_order_received_screen(uint16_t order_id) {
    lcd_clear();
    lcd_goto(0, 0);
    lcd_print_fixed_16("Order sent");
    lcd_goto(1, 0);
    lcd_puts("Order #");
    lcd_print_uint16(order_id);
}

void lcd_show_order_accepted_screen(uint16_t order_id) {
    lcd_clear();
    lcd_goto(0, 0);
    lcd_print_fixed_16("Order accepted");
    lcd_goto(1, 0);
    lcd_puts("Order #");
    lcd_print_uint16(order_id);
}

void lcd_show_order_ready_screen(uint16_t order_id) {
    lcd_clear();
    lcd_goto(0, 0);
    lcd_print_fixed_16("Order ready!");
    lcd_goto(1, 0);
    lcd_puts("Order #");
    lcd_print_uint16(order_id);
}

void lcd_show_orders_cleared_screen(void) {
    lcd_clear();
    lcd_goto(0, 0);
    lcd_print_fixed_16("Orders cleared");
    lcd_goto(1, 0);
    lcd_print_fixed_16("*=back");
}

void lcd_show_error_screen(const char *msg) {
    lcd_clear();
    lcd_goto(0, 0);
    lcd_print_fixed_16("Error");
    lcd_goto(1, 0);
    lcd_print_fixed_16(msg);
}

void lcd_show_main_menu(void) {
    lcd_clear();
    lcd_goto(0, 0);
    lcd_print_fixed_16("A=Meal B=Lunch");
    lcd_goto(1, 0);
    lcd_print_fixed_16("C=Drink D=Order");
}

void lcd_show_empty_cart(void) {
    lcd_clear();
    lcd_goto(0, 0);
    lcd_print_fixed_16("Cart is empty");
    lcd_goto(1, 0);
    lcd_print_fixed_16("*=Back");
}

void lcd_show_added_to_cart(const char *name, uint8_t qty) {
    lcd_clear();
    lcd_goto(0, 0);
    lcd_print_fixed_16("Added to cart");
    lcd_goto(1, 0);
    lcd_putc('x');
    if (qty == 10) {
        lcd_puts("10 ");
    } else {
        lcd_putc((char)('0' + qty));
        lcd_putc(' ');
    }
    lcd_print_fixed_16(name);
}

void lcd_show_cart_item(uint8_t index, uint8_t count, const char *name, uint8_t qty, uint16_t line_total) {
    lcd_clear();

    lcd_goto(0, 0);
    lcd_print_uint16(index + 1);
    lcd_putc('/');
    lcd_print_uint16(count);
    lcd_putc(' ');
    lcd_print_fixed_16(name);

    lcd_goto(1, 0);
    lcd_putc('x');
    lcd_print_uint16(qty);
    lcd_putc(' ');
    lcd_print_price(line_total);
    lcd_puts(" #=Send");
}