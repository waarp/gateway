=====================
Lister les transferts
=====================

.. program:: waarp-gateway transfer list

.. describe:: waarp-gateway <ADDR> transfer list

Affiche une liste des transferts remplissant les critères ci-dessous.

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Le nombre de maximum de transferts à afficher. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Les `n` premiers transferts de la liste seront ignoré.

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
   ``RUNNING``, ``INTERRUPTED`` & ``PAUSED``.

.. option:: -d <DATE>, --date=<DATE>

   Filtre les transferts ultérieurs à la date renseignée avec ce paramètre.
   La date doit être renseignée en suivant le format standard ISO 8601 tel
   qu'il est décrit dans la `RFC3339 <https://www.ietf.org/rfc/rfc3339.txt>`_.

|

**Exemple**

.. code-block:: bash

   waarp-gateway http://user:password@localhost:8080 transfer list -l 10 -o 5 -s id- -r règle_1 -t PLANNED -d 2019-01-01T12:00:00+02:00