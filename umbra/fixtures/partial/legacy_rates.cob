      * A COBOL rate table. Graph indexes COBOL inventory-only: it discovers
      * the file and no relations are extracted from it, so nothing here can
      * ever be confirmed structural evidence. It is in the fixture to give
      * the report a language tier to disclose.
       IDENTIFICATION DIVISION.
       PROGRAM-ID. LEGACY-RATES.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-TAX-RATE      PIC 9V999 VALUE 0.100.
       01 WS-DISCOUNT      PIC 9(4)V99 VALUE 5.00.
       PROCEDURE DIVISION.
       COMPUTE-TOTAL.
           DISPLAY "rates are read by the pricing module".
           STOP RUN.
