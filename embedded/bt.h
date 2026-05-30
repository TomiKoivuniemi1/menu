#ifndef BT_H
#define BT_H

#include <stdint.h>

void bt_init(void);
void bt_putc(char c);
void bt_puts(const char *s);
uint8_t bt_available(void);
uint8_t bt_read_byte(uint8_t *out);

#endif
