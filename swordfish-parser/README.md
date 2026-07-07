# swordfish-verifier/parser

Модуль **парсера спецификации** для системы верификации API СХД на
соответствие спецификации SNIA Swordfish

## Структура

```
model/            общая, независимая от формата модель Spec/Resource/Property
v1json/           Версия 1: парсер ТОЛЬКО для JSON Schema (Redfish/Swordfish)
v2universal/       Версия 2: универсальный парсер (интерфейс + диспетчер форматов)
v2universal/formats/
                   реализации FormatParser
testdata/          файлы для тестов
```