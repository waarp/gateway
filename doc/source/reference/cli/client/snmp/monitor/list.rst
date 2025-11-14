=========================
Lister les moniteurs SNMP
=========================

.. program:: waarp-gateway snmp monitor list

Affiche une liste des moniteurs SNMP connus.

**Commande**

.. code-block:: shell

   waarp-gateway snmp monitor list

**Options**

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Le nombre maximum de moniteurs autorisés dans la réponse. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Le numéro du premier moniteur renvoyé.

.. option:: -s <SORT>, --sort=<SORT>

   Le paramètre et l'ordre selon lesquels les moniteurs seront triés. Les
   choix possibles sont:

   - par nom (``name+`` & ``name-``)
   - par adresse (``address+`` et ``address-``)

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway snmp monitor list -l 10 -o 5 -s "name+"
