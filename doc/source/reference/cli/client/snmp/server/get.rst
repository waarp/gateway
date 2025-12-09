=========================================
Afficher la configuration du serveur SNMP
=========================================

.. program:: waarp-gateway snmp server get

Affiche la configuration du serveur SNMP de la Gateway.

**Commande**

.. code-block:: shell

   waarp-gateway snmp server get

**Options**

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway snmp server get
