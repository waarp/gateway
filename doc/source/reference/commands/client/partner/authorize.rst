=====================================
Autoriser une règle sur un partenaire
=====================================

.. program:: waarp-gateway partner authorize

.. describe:: waarp-gateway <ADDR> partner authorize <PARTNER> <RULE>

Autorise le partenaire demandé à utiliser la règle donnée. Les noms du partenaire
et de la règle doivent être fournis en argument de programme après la commande.

|

**Exemple**

.. code-block:: shell

   waarp-gateway http://user:password@localhost:8080 partner authorize waarp_sftp règle_1