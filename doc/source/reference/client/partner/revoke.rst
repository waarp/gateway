=====================================
Interdire une règle sur un partenaire
=====================================

.. program:: waarp-gateway partner revoke

.. describe:: waarp-gateway <ADDR> partner revoke <PARTNER> <RULE>

Retire au partenaire fourni en argument la permission d'utiliser la règle donnée.

|

**Exemple**

.. code-block:: shell

   waarp-gateway http://user:password@localhost:8080 partner revoke waarp_sftp règle_1