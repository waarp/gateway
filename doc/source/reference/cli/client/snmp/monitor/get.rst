=========================
Afficher un moniteur SNMP
=========================

.. program:: waarp-gateway snmp monitor get

Affiche les informations du moniteur SNMP donné en paramètre.

**Commande**

.. code-block:: shell

   waarp-gateway snmp monitor get "<NAME>"

**Options**

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway snmp monitor get "nagios"
