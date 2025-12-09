.. _reference-cli-client-transfers-list:

=====================
Lister les transferts
=====================

.. program:: waarp-gateway transfer list

Affiche une liste des transferts remplissant les critères ci-dessous.

**Commande**

.. code-block:: shell

   waarp-gateway transfer list

**Options**

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Le nombre de maximum de transferts à afficher. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Les ``n`` premiers transferts de la liste seront ignoré.

.. option:: -s <SORT>, --sort=<SORT>

   L'ordre et le paramètre selon lesquels les transferts seront triés. Les choix
   possibles sont :

   - tri par date (``start+`` & ``start-``)
   - tri par identifiant (``id+`` & ``id-``)
   - tri par statut (``status+`` & ``status-``)

.. option:: -r <RULE>, --rule_=<RULE>

   Filtre les transferts utilisant la règle renseigné avec ce paramètre.
   Le paramètre peut être renseigné plusieurs fois pour filtrer plusieurs
   règles à la fois.

.. option:: -t <STATUS>, --status=<STATUS>

   Filtre les transferts ayant actuellement le statut renseigné avec ce
   paramètre. Le paramètre peut être renseigné plusieurs fois pour filtrer
   plusieurs statuts à la fois. Les statuts valides sont: ``PLANNED``,
   ``RUNNING``, ``INTERRUPTED`` et ``PAUSED``.

.. option:: -d <DATE>, --date=<DATE>

   Filtre les transferts ultérieurs à la date renseignée avec ce paramètre.
   La date doit être renseignée en suivant le format standard ISO 8601 tel
   qu'il est décrit dans la rfc:`3339`.

.. option:: -f <FOLLOW_ID>, --follow-id=<FOLLOW_ID>

   Filtre les transferts ayant l'ID de flux renseigné avec ce paramètre. Les
   transferts renvoyés feront donc partie du même flux.

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway transfer list -l 10 -o 5 -s id- -r 'règle_1' -t 'PLANNED' -d '2019-01-01T12:00:00+02:00' -f '12345'
