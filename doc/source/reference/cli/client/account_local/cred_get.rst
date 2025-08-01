========================================
Consulter une méthode d'authentification
========================================

.. program:: waarp-gateway account local credential get

Affiche les informations de la méthode d'authentification donnée.

**Commande**

.. code-block:: shell

   waarp-gateway partner credential "<SERVER>" get "<CREDENTIAL>"

**Options**

.. option:: -r, --raw

   Affiche la valeur brute de la méthode d'authentification au lieu de ses
   métadonnées quand applicable. Par exemple, utiliser cette option sur un
   certificat TLS affichera le fichier PEM du certificat au lieu des informations
   du certificat.

**Exemple**

.. code-block:: shell

   waarp-gateway account local "sftp_server" credential "openssh" get "openssh_hostkey"
