.. _reference-cmd-waarp-gatewayd-purge:

########################
``waarp-gatewayd purge``
########################

.. program:: waarp-gatewayd purge

``waarp-gatewayd purge`` est la commande permettant de purger l'historique de
transfert afin de libérer de l'espace disque sur la base de données.

.. warning:: Évidemment, cette purge est irréversible, soyez donc prudent lorsque
   vous l'utilisez. La commande vous demandera confirmation avant d'effectuer la
   purge.

Cette commande accepte les options suivantes :

.. option:: --config FILE, -c FILE

   Définit le fichier de configuration à utiliser.

   Si aucun fichier spécifique n'est fourni avec cet argument, les emplacements
   par défaut suivants sont recherchés (dans cet ordre) :

   * :file:`gatewayd.ini`, relatif au dossier courant (Linux et Windows)
   * :file:`etc/gatewayd.ini`, relatif au dossier courant (Linux)
   * :file:`etc\\gatewayd.ini`, relatif au dossier courant (Windows)
   * :file:`/etc/waarp-gateway/gatewayd.ini` (Linux)
   * :file:`%ProgramData%\\gatewayd.ini` (Windows)

.. _reference-cmd-waarp-gatewayd-purge-older-than:

.. option:: --older-than, -o

   Limite la purge aux transferts plus anciens que le temps donné. Peut
   être soit une date, auquel cas, seuls les transferts antérieurs à cette date
   seront supprimés; soit une durée, auquel cas, seuls les transferts plus anciens
   que cette durée seront supprimés.

   La date doit être au format `aaaa/MM/dd hh:mm:ss` (exemple: `2022/10/25 08:00:00`).

   La durée est une succession arbitraire de valeurs de temps (exemple: `1y3mo2d`
   pour 1 an, 3 mois et 2 jours). Les unités de temps suivantes sont acceptées:

   * *Nanosecondes*: ns
   * *Microsecondes*: us, µs (U+00B5), μs (U+03BC)
   * *Millisecondes*: ms
   * *Secondes*: s, sec, second, seconds
   * *Minutes*: m, min, minute, minutes
   * *Heures*: h, hr, hour, hours
   * *Jours*: d, day, days
   * *Semaines*: w, wk, week, weeks
   * *Mois*: mo, mon, month, months
   * *Années*: y, yr, year, years

.. option:: --reset, -r

   Si cette option est présente, en plus de purger d'historique, la commande
   réinitialisera l'auto-incrément des identifiants locaux de transfert à zéro.

   .. warning:: Cette option ne peut être utilisée que si la table des transferts
      en cours est vide. Dans le cas contraire, la commande échouera.

.. option:: --verbose, -v

   Active l'écriture des logs sur la sortie d'erreur.
   Cet argument peut être répété jusqu'à 3 fois pour augmenter la verbosité
   (ex : ``-vvv``).