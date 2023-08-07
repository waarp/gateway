====================
Retenter un transfer
====================

.. program:: waarp-gateway history retry

Retente le transfert demandé. L'ID du transfert doit être fournit en
argument de commande. Seuls les transferts ayant échoué peuvent être retentés.

.. option:: -d <DATE>, --date=<DATE>

   La date à laquelle le transfert redémarrera. Par défaut, le transfert
   redémarre immédiatement. La date doit être renseignée en suivant le
   format standard ISO 8601 tel qu'il est décrit dans la
   :rfc:`3339`.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' history restart 1234
