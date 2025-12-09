========================================
Consulter une méthode d'authentification
========================================

.. program:: waarp-gateway account remote credential get

Affiche les informations de la méthode d'authentification donnée.

**Commande**

.. code-block:: shell

   waarp-gateway account remote "<PARTNER>" credential "<LOGIN>" get "<CREDENTIAL>"

**Options**

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway account remote "openssh" credential "waarp_ssh" get "password"
