======================================
Ajouter une méthode d'authentification
======================================

.. program:: waarp-gateway server credential get

.. describe:: waarp-gateway server <SERVER> credential get

Ajoute une nouvelle valeur d'authentification au serveur donné. Si une valeur du
même nom existe déjà, elle sera écrasée.

.. option:: -n <NAME>, --name=<NAME>

   Le nom de la valeur. Par défaut, le type est utilisé comme nom.

.. option:: -t <AUTH_TYPE>, --type=<AUTH_TYPE>

   Le type d'authentification. Voir la :ref:`liste des méthodes d'authentification
   <reference-auth-methods>` pour la liste des différents types d'authentification.
   Pour les serveurs locaux, une méthode d'authentification externe est requise.

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

   waarp-gateway -a 'http://user:password@localhost:8080' server credential 'gw_r66' get -n 'r66_cert' -t 'ssh_private_key' -v './gw_ssh.key'
