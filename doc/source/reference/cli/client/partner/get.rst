======================
Afficher un partenaire
======================

.. program:: waarp-gateway partner get

Affiche les informations du partenaire donné en paramètre.

**Commande**

.. code-block:: shell

   waarp-gateway partner get "<PARTNER>"

**Options**

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway partner get 'partenaire_sftp'
