=====================================
Interdire une règle sur un partenaire
=====================================

.. program:: waarp-gateway partner revoke

Retire au partenaire fourni en argument la permission d'utiliser la règle donnée.

**Commande**

.. code-block:: shell

   waarp-gateway partner revoke "<PARTNER>" "<RULE>"

**Exemple**

.. code-block:: shell

   waarp-gateway partner revoke 'waarp_sftp' 'règle_1'
