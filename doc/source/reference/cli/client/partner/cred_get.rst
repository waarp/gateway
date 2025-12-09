========================================
Consulter une méthode d'authentification
========================================

.. program:: waarp-gateway partner credential get

Affiche les informations de la méthode d'authentification donnée.

**Commande**

.. code-block:: shell

   waarp-gateway partner credential "<PARTNER>" get "<CREDENTIAL>"

**Options**

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway partner credential "openssh" get "openssh_hostkey"
