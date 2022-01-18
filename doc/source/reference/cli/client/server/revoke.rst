==================================
Interdire une règle sur un serveur
==================================

.. program:: waarp-gateway server revoke

.. describe:: waarp-gateway server revoke <SERVER> <RULE>

Retire au serveur fourni en paramètre la permission d'utiliser la règle donnée.

|

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' server revoke 'gw_r66' 'règle_1'