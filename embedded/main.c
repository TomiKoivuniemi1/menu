#ifndef F_CPU
#define F_CPU 16000000UL
#endif

#include <util/delay.h>
#include <stdint.h>
#include <string.h>
#include <stdio.h>
#include <stdlib.h>

#include "bt.h"
#include "lcd.h"
#include "keypad.h"
#include "buzzer.h"

#define CAT_DRINKS 1
#define CAT_LUNCH  2
#define CAT_MEAL   3

typedef struct {
    uint8_t id;
    uint8_t category;
    char name[17];
    uint16_t price;
    uint8_t available;
} MenuItem;

typedef struct {
    uint8_t item_id;
    const char *name;
    uint8_t qty;
    uint16_t price;
} CartLine;

typedef enum {
    SCREEN_MAIN = 0,
    SCREEN_MEAL,
    SCREEN_LUNCH,
    SCREEN_DRINKS,
    SCREEN_CART
} Screen;

#define CART_MAX 10
#define MENU_MAX 50
#define RX_BUF_SIZE 96

static CartLine cart[CART_MAX];
static uint8_t cart_count = 0;
static MenuItem menu_items[MENU_MAX];
static uint8_t menu_count = 0;

static uint8_t screen_match(Screen screen, uint8_t cat) {
    if (screen == SCREEN_MEAL) return cat == CAT_MEAL;
    if (screen == SCREEN_LUNCH) return cat == CAT_LUNCH;
    if (screen == SCREEN_DRINKS) return cat == CAT_DRINKS;
    return 0;
}

static uint8_t parse_category(const char *cat) {
    if (strcmp(cat, "Drinks") == 0) return CAT_DRINKS;
    if (strcmp(cat, "Lunch") == 0) return CAT_LUNCH;
    if (strcmp(cat, "Meal") == 0) return CAT_MEAL;
    return 0;
}

static MenuItem *get_item_by_index(Screen screen, uint8_t index, uint8_t *count) {
    uint8_t seen = 0;
    MenuItem *result = 0;

    *count = 0;

    for (uint8_t i = 0; i < menu_count; i++) {
        if (!menu_items[i].available) continue;
        if (!screen_match(screen, menu_items[i].category)) continue;

        if (seen == index) {
            result = &menu_items[i];
        }

        seen++;
    }

    *count = seen;
    return result;
}

static uint16_t parse_short_id(const char *line) {
    uint16_t value = 0;
    line++;

    while (*line >= '0' && *line <= '9') {
        value = (uint16_t)((value * 10) + (*line - '0'));
        line++;
    }

    return value;
}

static void show_current_item(Screen screen, uint8_t index, uint8_t qty) {
    uint8_t count = 0;
    MenuItem *item = get_item_by_index(screen, index, &count);

    if (!item || count == 0) {
        lcd_show_error_screen("No menu items");
        return;
    }

    lcd_show_item_screen(item->name, item->price, qty);
}

static uint16_t cart_total(void) {
    uint16_t total = 0;

    for (uint8_t i = 0; i < cart_count; i++) {
        total += cart[i].price * cart[i].qty;
    }

    return total;
}

static void show_cart(uint8_t index) {
    if (cart_count == 0) {
        lcd_show_empty_cart();
        return;
    }

    if (index >= cart_count + 1) index = 0;

    if (index == cart_count) {
        lcd_show_checkout_screen(cart_total());
        return;
    }

    lcd_show_cart_item(
        index,
        cart_count,
        cart[index].name,
        cart[index].qty,
        cart[index].price * cart[index].qty
    );
}

static void add_to_cart(uint8_t id, const char *name, uint16_t price, uint8_t qty) {
    for (uint8_t i = 0; i < cart_count; i++) {
        if (cart[i].item_id == id) {
            cart[i].qty += qty;
            if (cart[i].qty > 99) cart[i].qty = 99;
            return;
        }
    }

    if (cart_count >= CART_MAX) return;

    cart[cart_count].item_id = id;
    cart[cart_count].name = name;
    cart[cart_count].price = price;
    cart[cart_count].qty = qty;
    cart_count++;
}

static void send_order(void) {
    if (cart_count == 0) {
        lcd_show_empty_cart();
        return;
    }

    bt_puts("ORDER|TABLE1|");

    for (uint8_t i = 0; i < cart_count; i++) {
        char buf[12];

        if (i > 0) {
            bt_putc(',');
        }

        sprintf(buf, "%u:%u", cart[i].item_id, cart[i].qty);
        bt_puts(buf);
    }

    bt_puts("|MENU_DEVICE\r\n");

    lcd_show_order_sent_screen();
}

static void handle_bt_line(const char *line) {
    if (line[0] == 'V') {
        lcd_show_order_received_screen(parse_short_id(line));
        return;
    }

    if (line[0] == 'A') {
        lcd_show_order_accepted_screen(parse_short_id(line));
        return;
    }

    if (line[0] == 'R') {
        lcd_show_order_ready_screen(parse_short_id(line));
        buzzer_ready_tune();
        cart_count = 0;
        return;
    }

    if (strcmp(line, "C") == 0) {
        cart_count = 0;
        lcd_show_orders_cleared_screen();
        return;
    }

    if (strcmp(line, "MENU_BEGIN") == 0) {
    menu_count = 0;
    return;
    }

    if (strncmp(line, "MENU_ITEM|", 10) == 0) {
        if (menu_count >= MENU_MAX) {
            return;
        }

        char copy[RX_BUF_SIZE];
        strncpy(copy, line, sizeof(copy) - 1);
        copy[sizeof(copy) - 1] = '\0';

        char *token = strtok(copy, "|");

        token = strtok(NULL, "|");
        if (!token) return;
        menu_items[menu_count].id = (uint8_t)atoi(token);

        token = strtok(NULL, "|");
        if (!token) return;
        menu_items[menu_count].category = parse_category(token);

        token = strtok(NULL, "|");
        if (!token) return;
        strncpy(menu_items[menu_count].name, token, sizeof(menu_items[menu_count].name) - 1);
        menu_items[menu_count].name[sizeof(menu_items[menu_count].name) - 1] = '\0';

        token = strtok(NULL, "|");
        if (!token) return;
        menu_items[menu_count].price = (uint16_t)atoi(token);

        token = strtok(NULL, "|");
        if (!token) return;
        menu_items[menu_count].available = (uint8_t)atoi(token);

        menu_count++;
        return;
    }

    if (strcmp(line, "MENU_END") == 0) {
        return;
    }
    if (strcmp(line, "MENU_CHANGED") == 0) {
        lcd_clear();
        lcd_goto(0, 0);
        lcd_print_fixed_16("Menu updated");
        lcd_goto(1, 0);
        lcd_print_fixed_16("*=Back");
        return;
    }

    if (strncmp(line, "ORDER_ERROR|", 12) == 0) {
        lcd_show_error_screen("Order failed");
        return;
    }
}

static void poll_bt(void) {
    static char rx_buf[RX_BUF_SIZE];
    static uint8_t rx_len = 0;

    uint8_t ch = 0;

    while (bt_read_byte(&ch)) {
        if (ch == '\r') {
            continue;
        }

        if (ch == '\n') {
            rx_buf[rx_len] = '\0';

            if (rx_len > 0) {
                handle_bt_line(rx_buf);
            }

            rx_len = 0;
            continue;
        }

        if (rx_len < RX_BUF_SIZE - 1) {
            rx_buf[rx_len++] = (char)ch;
        } else {
            rx_len = 0;
        }
    }
}

int main(void) {
    Screen screen = SCREEN_MAIN;
    uint8_t item_index = 0;
    uint8_t cart_index = 0;
    uint8_t qty = 1;

    bt_init();
    lcd_init();
    keypad_init();
    buzzer_init();

    lcd_show_main_menu();

    bt_puts("MENU_DEVICE_BOOT\r\n");
    bt_puts("HELLO\r\n");

    while (1) {
        poll_bt();

        char key = keypad_getkey();

        if (key) {
                if (key == '*') {
                    screen = SCREEN_MAIN;
                    lcd_show_main_menu();
                    continue;
                }

                if (screen == SCREEN_MAIN) {
                    if (key == 'A') {
                    screen = SCREEN_MEAL;
                    item_index = 0;
                    qty = 1;
                    show_current_item(screen, item_index, qty);
                } else if (key == 'B') {
                    screen = SCREEN_LUNCH;
                    item_index = 0;
                    qty = 1;
                    show_current_item(screen, item_index, qty);
                } else if (key == 'C') {
                    screen = SCREEN_DRINKS;
                    item_index = 0;
                    qty = 1;
                    show_current_item(screen, item_index, qty);
                } else if (key == 'D') {
                    screen = SCREEN_CART;
                    cart_index = 0;
                    show_cart(cart_index);
                }
            } else if (screen == SCREEN_CART) {
                if (key == '*') {
                    screen = SCREEN_MAIN;
                    lcd_show_main_menu();
                } else if (key == 'A') {
                    if (cart_count > 0) {
                        if (cart_index == 0) cart_index = cart_count;
                        else cart_index--;
                    }
                    show_cart(cart_index);
                } else if (key == 'B') {
                    if (cart_count > 0) {
                        cart_index++;
                        if (cart_index >= cart_count + 1) cart_index = 0;
                    }
                    show_cart(cart_index);
                } else if (key == '#') {
                    send_order();
                }
            } else {
                uint8_t count = 0;
                MenuItem *item = get_item_by_index(screen, item_index, &count);
                if (count == 0) {
                    lcd_show_error_screen("No menu items");
                    continue;
                }

                if (key == '*') {
                    screen = SCREEN_MAIN;
                    lcd_show_main_menu();
                } else if (key == 'A') {
                    if (item_index == 0) item_index = count - 1;
                    else item_index--;
                    qty = 1;
                    show_current_item(screen, item_index, qty);
                } else if (key == 'B') {
                    item_index++;
                    if (item_index >= count) item_index = 0;
                    qty = 1;
                    show_current_item(screen, item_index, qty);
                } else if (key >= '1' && key <= '9') {
                    qty = (uint8_t)(key - '0');
                    show_current_item(screen, item_index, qty);
                } else if (key == '0') {
                    qty = 10;
                    show_current_item(screen, item_index, qty);
                } else if (key == '#') {
                    if (item) {
                        add_to_cart(item->id, item->name, item->price, qty);
                        lcd_show_added_to_cart(item->name, qty);
                    }
                    _delay_ms(900);
                    show_current_item(screen, item_index, qty);
                }
            }
        }

        _delay_ms(1);
    }
}