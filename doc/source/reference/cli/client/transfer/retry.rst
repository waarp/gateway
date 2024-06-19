====================
Retenter un transfer
====================

.. program:: waarp-gateway transfer retry

Rejoue le transfert demandé en intégralité. L'ID du transfert doit être fournit en
argument de commande. Seuls les transferts ayant terminé peuvent être rejoués.

**Commande**

.. code-block:: shell

   waarp-gateway transfer retry "<TRANSFER_ID>"

**Options**

.. option:: -d <DATE>, --date=<DATE>

   La date à laquelle le transfert redémarrera. Par défaut, le transfert
   redémarre immédiatement. La date doit être renseignée en suivant le format
   standard ISO 8601 tel qu'il est décrit dans la :rfc:`3339`.

**Exemple**

.. code-block:: shell

   waarp-gateway transfer retry 1234 -d "2021-01-01T01:00:00+02:00"
