###########################
Codes d'erreur de transfert
###########################



:index:`TeOk`
   C'est le code utilisé par défaut. Il indique qu'il n'y a aucune erreur

:index:`TeUnknown`
   Indique une erreur inconnue (les details de l'erreur peuvent donner plus
   d'informations).

:index:`TeInternal`
   Indique qu'une erreur interne à la gateway s'est produite (perte de
   connection à la base de données, etc.)

:index:`TeUnimplemented`
   Indique que le partenaire a voulu utiliser une fonctionnalité valide,, mais
   qui n'est pas implémentée par la passerelle.

:index:`TeConnection`
   Indique que la connexion à un partenaire distant ne peut être établie. Il
   s'agit du code d'erreur le plus généraliste. Un code plus spécifique, s'il
   existe, doit être utilisé.

:index:`TeConnectionReset`
   means "connection reset by peer"

:index:`TeUnknownRemote`
   Indique que le partenaire distant demandé n'est pas connu dans la base de
   données de la gateway. Pour des raisons de sécurité, l'utilisation externe de
   ce code d'erreur (retourner cette erreur a un client) est découragé pour les
   connexions entrantes au profit de ``TeBadAuthentication``, au risque de
   laisser fuiter des informations (ici, le compte existe, mais
   l'authentification a echouée).

:index:`TeExceededLimit`
   Indique que le transfert ne peut être traité parce que les limites définies
   ont été atteintes (nombre de connexions, nombre de transferts, etc.)

:index:`TeBadAuthentication`
   Indique une erreur de connexion causée par des données d'authentification
   erronées (utilisateur, mot de passe, certificat, etc. les détails de l'erreur
   peuvent fournir plus d'informations).

:index:`TeDataTransfer`
   Indique qu'une erreur s'est produite durant le transfert des données. Il
   s'agit d'une erreur générale et un code plus précis, s'il existe, doit être
   utilisé.

:index:`TeIntegrity`
   Indique que la vérification de l'intégrité a échoué.

:index:`TeFinalization`
   Indique qu'une erreur s'est produite durant la finalisation du transfert.

:index:`TeExternalOperation`
   Indique qu'une erreur s'est produite durant l'exécution des traitements
   pré-transfert, post-transfert, ou d'erreur.

:index:`TeStopped`
   Indique que le transfert a été arrêté.

:index:`TeCanceled`
   Indique que le transfert a été annulé.

:index:`TeFileNotFound`
   Indique que le fichier demandé n'a pas été trouvé.

:index:`TeForbidden`
   Indique que le partenaire distant n'a pas le droit d'effectuer une action

:index:`TeBadSize`
   Indique une erreur liée à la taille du fichier (elle dépasse un quota, il n'y
   a plus assez d'espace que le disque de destination, etc.)

:index:`TeShuttingDown`
   Indique que la passerelle est en cours d'arrêt

