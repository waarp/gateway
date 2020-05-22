==================================
Interdire une règle sur un serveur
==================================

.. program:: waarp-gateway server revoke

.. describe:: waarp-gateway <ADDR> server revoke <SERVER> <RULE>

Retire au serveur fourni en paramètre la permission d'utiliser la règle donnée.

|

**Exemple**

.. code-block:: bash

   waarp-gateway http://user:password@localhost:8080 server revoke serveur_sftp règle_1