==================================
Autoriser une règle sur un serveur
==================================

.. program:: waarp-gateway server authorize

.. describe:: waarp-gateway server authorize <SERVER> <RULE>

Autorise le serveur fourni en paramètre à utiliser la règle donnée.

|

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' server authorize 'gw_r66' 'règle_1'