#ifndef LCD_H
#define LCD_H

#include <stdint.h>

void lcd_init(void);
void lcd_clear(void);
void lcd_goto(uint8_t row, uint8_t col);
void lcd_putc(char c);
void lcd_puts(const char *s);
void lcd_puts_padded(const char *s);
void lcd_show_home_screen(void);
void lcd_show_category_screen(void);
void lcd_show_item_screen(const char *name, uint16_t price_cents, uint8_t qty);
void lcd_show_quantity_screen(const char *name, uint8_t qty);
void lcd_show_cart_screen(uint8_t item_count, uint16_t total_cents);
void lcd_show_checkout_screen(uint16_t total_cents);
void lcd_show_order_sent_screen(void);
void lcd_show_order_received_screen(uint16_t order_id);
void lcd_show_order_accepted_screen(uint16_t order_id);
void lcd_show_order_ready_screen(uint16_t order_id);
void lcd_show_orders_cleared_screen(void);
void lcd_show_error_screen(const char *msg);
void lcd_print_price(uint16_t price_cents);
void lcd_print_fixed_16(const char *text);
void lcd_show_main_menu(void);
void lcd_show_cart_item(uint8_t index, uint8_t count, const char *name, uint8_t qty, uint16_t line_total);
void lcd_show_empty_cart(void);
void lcd_show_added_to_cart(const char *name, uint8_t qty);

#endif
