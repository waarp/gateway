====================
Retenter un transfer
====================

.. program:: waarp-gateway transfer retry

.. describe:: waarp-gateway transfer retry <TRANS>

Retente le transfert demandé. L'ID du transfert doit être fournit en
argument de commande. Seuls les transferts ayant terminé peuvent être retentés.

.. option:: -d <DATE>, --date=<DATE>

   La date à laquelle le transfert redémarrera. Par défaut, le transfert
   redémarre immédiatement. La date doit être renseignée en suivant le
   format standard ISO 8601 tel qu'il est décrit dans la
   `RFC3339 <https://www.ietf.org/rfc/rfc3339.txt>`_.

|

**Exemple**

.. code-block:: shell

   waarp-gateway http://user:password@localhost:8080 transfer restart 1234 -d 2021-01-01T01:00:00+02:00