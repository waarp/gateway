======================================
Ajouter une méthode d'authentification
======================================

.. program:: waarp-gateway account remote auth add

.. describe:: waarp-gateway account remote <PARTNER> auth <LOGIN> add

Ajoute une nouvelle valeur d'authentification au compte local donné. Si une
valeur du même nom existe déjà, elle sera écrasée.

.. option:: -n <NAME>, --name=<NAME>

   Le nom de la valeur. Par défaut, le type est utilisé comme nom.

.. option:: -t <AUTH_TYPE>, --type=<AUTH_TYPE>

   Le type d'authentification. Voir la :ref:`liste des méthodes d'authentification
   <reference-auth-methods>` pour la liste des différents types d'authentification.
   Pour les comptes locaux, une méthode d'authentification interne est requise.

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

   waarp-gateway -a 'http://user:password@localhost:8080' account remote 'openssh' auth 'titi' add -t 'password' -v 'sesame'
