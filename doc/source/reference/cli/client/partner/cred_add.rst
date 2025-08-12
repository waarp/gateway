======================================
Ajouter une méthode d'authentification
======================================

.. program:: waarp-gateway partner credential add

Ajoute une nouvelle valeur d'authentification au partenaire donné. Si une valeur
du même nom existe déjà, elle sera écrasée.

**Commande**

.. code-block:: shell

   waarp-gateway partner credential "<PARTNER>" add

**Options**

.. option:: -n <NAME>, --name=<NAME>

   Le nom de la valeur. Par défaut, le type est utilisé comme nom.

.. option:: -t <AUTH_TYPE>, --type=<AUTH_TYPE>

   Le type d'authentification. Voir la :ref:`liste des méthodes d'authentification
   <reference-auth-methods>` pour la liste des différents types d'authentification.
   Pour les partenaires distants, une méthode d'authentification interne est requise.

.. option:: -v <VALUE>, --value=<VALUE>

   La valeur d'authentification (le mot de passe, le certificat...). Cette option
   accepte également les chemins de fichiers, auquel cas, le contenu du fichier
   donné sera utilisé comme valeur.

.. option:: -s <VALUE>, --secondary-value=<VALUE>

   La valeur secondaire d'authentification (pour les méthodes où cela est nécessaire,
   voir la :ref:`liste des méthodes d'authentification<reference-auth-methods>` pour
   savoir quand cela est requis). Similairement à l'option ``-v`` ci-dessus, cette
   option accepte les chemins de fichiers pour renseigner la valeur.


**Exemple**

.. code-block:: shell

   waarp-gateway partner credential "openssh" add -n "openssh_hostkey" -t "ssh_public_key" -v "./ssh.pub"
