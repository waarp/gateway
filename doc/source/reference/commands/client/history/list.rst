=====================
Afficher l'historique
=====================

.. program:: waarp-gateway transfer list

.. describe:: waarp-gateway <ADDR> history list

Affiche une liste des entrées d'historique remplissant les critères ci-dessous.

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Le nombre de maximum d'entrées à afficher. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Le numéro de la première entrée renvoyée.

.. option:: -s <SORT>, --sort=<SORT>

   L'ordre et l'attributs selon lesquels les entrées seront triées. Les choix
   possibles sont:

   - tri par date (``start+`` & ``start-``)
   - tri par identifiant (``id+`` & ``id-``)
   - tri par origine du transfert (``requester+`` & ``requester-``)
   - tri par destination du transfert (``requested+`` & ``requested-``)
   - tri par règle (``rule+`` & ``rule-``)

.. option:: -q <ORIGIN>, --requester=<ORIGIN>

   Filtre les entrées selon le demandeur du transfert renseigné avec
   ce paramètre. Le paramètre peut être renseigné plusieurs fois pour
   filtrer plusieurs demandeurs à la fois.

.. option:: -d <DESTINATION>, --requested=<DESTINATION>

   Filtre les entrées selon le destinataire du transfert renseigné avec
   ce paramètre. Le paramètre peut être renseigné plusieurs fois pour
   filtrer plusieurs destinataires à la fois.

.. option:: -r <RULE>, --rule=<RULE>

   Filtre les entrées avec le nom de règle renseigné avec ce paramètre.
   Le paramètre peut être renseigné plusieurs fois pour filtrer plusieurs
   règles à la fois.

.. option:: -t <STATUS>, --status=<STATUS>

   Filtre les entrées ayant terminées le statut renseigné dans ce
   paramètre. Le paramètre peut être renseigné plusieurs fois pour filtrer
   plusieurs statuts à la fois.

.. option:: -p <PROTOCOL>, --protocol=<PROTOCOL>

   Filtre les entrée ayant ayant été effectuées avec le protocole
   renseigné dans ce paramètre. Le paramètre peut être renseigné plusieurs
   fois pour filtrer plusieurs statuts à la fois.

.. option:: -b <START>, --start=<START>

   Filtre les entrées ultérieures à la date renseignée avec ce paramètre.
   La date doit être renseignée en suivant le format standard ISO 8601 tel
   qu'il est décrit dans la `RFC3339 <https://www.ietf.org/rfc/rfc3339.txt>`_.

.. option:: -e <STOP>, --stop=<STOP>

   Filtre les entrées antérieures à la date renseignée avec ce paramètre.
   La date doit être renseignée en suivant le format standard ISO 8601 tel
   qu'il est décrit dans la `RFC3339 <https://www.ietf.org/rfc/rfc3339.txt>`_.

|

**Exemple**

.. code-block:: shell

   waarp-gateway http://user:password@localhost:8080 history list -l 10 -o 5 -s id- --requester=toto --requested=serveur_sftp --rule=règle_1 --protocol=sftp --status=DONE --start=2019-01-01T12:00:00+02:00 --stop=2019-01-02T12:00:00+02:00
